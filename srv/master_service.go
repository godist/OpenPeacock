package srv

import (
	"errors"
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/parallel"
	"github.com/wangkuiyi/phoenix/core/hist"
	"log"
	"net/rpc"
	"path"
	"regexp"
	"sync"
)

const (
	INIT = iota
	GIBBS
	LOGLL
)

var (
	InvalidReporter       = errors.New("Invalid reporter")
	TaskNotInWorkingQueue = errors.New("Completed task not in working queue")
)

type Master struct {
	cfg *Config

	// Master writes to finished after the trainining job is done, so
	// the creater of this channel could be notified of this event.
	finished chan bool

	// Master maintains three task queues, protected by mutex
	// schedule.
	schedule sync.Mutex
	pending  []*Task
	working  map[string]*Task // a task might be executed by multiple squads.

	// Aggregator information
	register    sync.Mutex
	aggregators []*RpcClient

	iteration int
}

func NewMaster(c *Config, finished chan bool) (*Master, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	m := &Master{
		cfg:         c,
		finished:    finished,
		pending:     make([]*Task, 0),
		working:     make(map[string]*Task),
		aggregators: make([]*RpcClient, 0, c.NumVShards),
		iteration:   -1,
	}
	if e := m.initializeTasks(); e != nil {
		return nil, e
	}
	return m, nil
}

// isCompletedIteration returns false if there is any error.
func isCompletedIteration(cfg *Config, iteration int) (bool, error) {
	for v := 0; v < cfg.NumVShards; v++ {
		f := path.Join(cfg.JobDir, fmt.Sprintf("%05d", iteration),
			fmt.Sprintf("%s-%05d-of-%05d",
				MODEL_FILE, v, cfg.NumVShards))
		if b, e := file.Exists(f); !b || e != nil {
			return false, e
		}
	}
	return true, nil
}

// FindMostRecentCompletedIteration returns 0 if it found a finished
// initialization iteration, 1 and larger for having found Gibbs
// sampling iterations, or -1 if no finished iteration was found.
func FindMostRecentCompletedIteration(cfg *Config) (int, error) {
	is, e := file.List(cfg.JobDir)
	if e != nil {
		return -1, fmt.Errorf("Failed to list %s: %v", cfg.JobDir, e)
	}

	iterationDir := regexp.MustCompile("^[0-9]+$")
	maxIter := -1

	for _, f := range is {
		if f.IsDir && iterationDir.MatchString(f.Name) {
			var iter int
			fmt.Sscanf(f.Name, "%05d", &iter)
			if iter > maxIter {
				if b, e := isCompletedIteration(cfg, iter); b && e == nil {
					maxIter = iter
				} else if e != nil {
					return -1, e
				}
			}
		}
	}

	return maxIter, nil
}

// initializeTasks is called when master starts/restarts or a new
// iteration starts.
func (m *Master) initializeTasks() error {
	fi, e := FindMostRecentCompletedIteration(m.cfg)
	if e != nil {
		return fmt.Errorf("FindMostRecentCompletedIteration: %v", e)
	}

	log.Printf("Initialize tasks for iteration %d", fi+1)
	m.iteration = fi + 1

	action := INIT
	if fi >= 0 && // with initialization or a sampling iteration done
		m.cfg.LogllPeriod > 0 && // log-likelihood is enabled
		fi%m.cfg.LogllPeriod == 0 { // it is time for logll
		action = LOGLL
	} else {
		if fi < 0 {
			action = INIT
		} else {
			action = GIBBS
		}
		fi += 1
		// Create directory for the next iteration.
		e := file.MkDir(path.Join(m.cfg.JobDir, fmt.Sprintf("%05d", fi)))
		if e != nil {
			return fmt.Errorf("Failed create directory %s: %v",
				path.Join(m.cfg.JobDir, fmt.Sprintf("%05d", fi)), e)
		}
	}

	is, e := file.List(m.cfg.CorpusDir)
	if e != nil {
		return fmt.Errorf("Failed list corpus dir %s: %v", m.cfg.CorpusDir, e)
	}
	var task *Task
	for i, info := range is {
		if i%m.cfg.NumVShards == 0 {
			if task != nil {
				m.pending = append(m.pending, task)
			}
			task = NewTask(m.cfg.NumVShards, fi, action)
		}
		task.Shards = append(task.Shards, info.Name)
	}
	m.pending = append(m.pending, task)

	return nil
}

// distributeTask issues a pending task, if there is any, to coordinator.
func (m *Master) distributeTask(coordinator string, task *Task) error {
	// If no more pending task, we know that an iteration is completed, so
	if len(m.pending) <= 0 {
		// If it is the initialization iteration finished, master
		// should help aggregators to aggregate their global topic
		// histograms.
		if e := m.aggregateGlobalHists(); e != nil {
			return e
		}
		// If there has been an iteration finished, master notify
		// aggregators to checkpoint model, and
		if e := m.saveModel(); e != nil {
			return e
		}
		// creates tasks for the new iteration, and return a pending
		// task of the new iteration.
		m.initializeTasks()
	}

	// TODO(wyi): we did not considered the case that some working
	// tasks halt for a long time.
	if len(m.pending) > 0 {
		m.pending[0].Coord = coordinator // Assign coordinator to task.
		*task = *(m.pending[0])
		m.working[coordinator] = m.pending[0]
		m.pending = m.pending[1:]
		return nil
	}
	return errors.New("Failed create tasks for new iteration")
}

func (m *Master) aggregateGlobalHists() error {
	var mutex sync.Mutex
	gh := hist.NewDense(m.cfg.NumTopics)

	if e := parallel.For(0, len(m.aggregators), 1, func(i int) error {
		h := hist.NewDense(m.cfg.NumTopics)
		var dumb int
		e := m.aggregators[i].Call("Aggregator.GetGlobalHist", &dumb, &h)
		if e != nil {
			return e
		}

		mutex.Lock()
		defer mutex.Unlock()
		h.ForEach(func(t int, c int64) error {
			gh.Inc(t, int(c))
			return nil
		})
		return nil
	}); e != nil {
		return fmt.Errorf("master aggregate global hists: %v", e)
	}

	return parallel.For(0, len(m.aggregators), 1, func(i int) error {
		return m.aggregators[i].Call("Aggregator.SetGlobalHist", gh, nil)
	})
}

func (m *Master) saveModel() error {
	return parallel.For(0, len(m.aggregators), 1, func(i int) error {
		e := m.aggregators[i].Call("Aggregator.Save", &struct {
			Iter, VShard, VShards int
		}{m.iteration, i, m.cfg.NumVShards}, nil)
		if e != nil {
			return fmt.Errorf("%s save model at iteration %d: %v",
				m.aggregators[i], 0, e)
		}
		return nil
	})
}

// RegisterSquad is supposed to be called by a coordinator that is
// just started or restarted to acquire its task.  The coordinator
// reports its address and gets a set of corpus shard files.  Note
// that the full path name of these shard files encodes the current
// iteration.
func (m *Master) RegisterSquad(coordinator string, task *Task) error {
	m.schedule.Lock()
	defer m.schedule.Unlock()

	// If coordinator identifies a working squad that was restarted, just
	// send it the task it was working on.
	if t, ok := m.working[coordinator]; ok {
		*task = *t
		return nil
	}

	// Otherwise, returns a task in the pending queue.
	return m.distributeTask(coordinator, task)
}

// CompleteTask is supposed to be called by a coordinator that
// finished a task.
func (m *Master) CompleteTask(did *Task, ret *Task) error {
	m.schedule.Lock()
	defer m.schedule.Unlock()

	for c, t := range m.working {
		if t.Equal(did) { // content (shard paths) are identical
			if did.Coord == c {
				delete(m.working, c)
				return m.distributeTask(did.Coord, ret)
			} else {
				return InvalidReporter
			}
		}
	}
	return TaskNotInWorkingQueue
}

func (m *Master) RegisterAggregator(aggr string, _ *int) error {
	m.register.Lock()
	defer m.register.Unlock()
	if c, e := rpc.DialHTTP("tcp", aggr); e != nil {
		return fmt.Errorf("Failed to dial aggregator %s: %v", aggr, e)
	} else {
		log.Printf("Aggregator %s registered", aggr)
		m.aggregators = append(m.aggregators, &RpcClient{c, aggr})
	}

	if len(m.aggregators) >= len(m.cfg.Aggregators) {
		log.Printf("Aggregtors all registered, starting squads.")
		if e := LaunchSquads(m.cfg); e != nil {
			KillSquads(m.cfg)
			return fmt.Errorf("Failed start squads: %v", e)
		}
	}
	return nil
}
