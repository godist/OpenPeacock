package srv

import (
	"fmt"
	"github.com/wangkuiyi/parallel"
	"net/rpc"
)

// Full path names of shard files under processing.  Note that squads
// always output to temporary files and rename them to output files
// after they were completely and successfully generated.  So we can
// determine if a task is completed by checking the existence of its
// output files.
type Task struct {
	Shards    []string
	Coord     string // the assignee, optional
	Iteration int
	Action    int // {INIT, GIBBS, LOGLL}
}

func NewTask(vshard int, iteration int, action int) *Task {
	return &Task{
		Shards:    make([]string, 0, vshard),
		Iteration: iteration,
		Action:    action,
	}
}

// Two tasks are equal to each other iff they have the same sequence
// of Shards and identical Action.
func (t *Task) Equal(o *Task) bool {
	if t == o {
		return true
	} else if t == nil || o == nil {
		return false
	} else if len(t.Shards) != len(o.Shards) {
		return false
	} else {
		if t.Action != o.Action {
			return false
		}
		for i, f := range t.Shards {
			if f != o.Shards[i] {
				return false
			}
		}
	}
	return true
}

// RpcClient represent rpc.Client and the address.  This makes it easy
// to display RPC connections in logs or expvars.
type RpcClient struct {
	*rpc.Client
	Name string
}

// String is required by interface Stringer.
func (r *RpcClient) String() string {
	return r.Name
}

func connectToSamplers(samplers []string) ([]*RpcClient, error) {
	clients := make([]*RpcClient, len(samplers))
	if e := parallel.For(0, len(samplers), 1, func(i int) error {
		if cl, e := rpc.DialHTTP("tcp", samplers[i]); e == nil {
			clients[i] = &RpcClient{cl, samplers[i]}
		} else {
			return fmt.Errorf("Connect to sampler %s: %v", samplers[i], e)
		}
		return nil
	}); e != nil {
		return nil, e
	}
	return clients, nil
}

func connectToAggregators(aggregators []string) ([]*RpcClient, error) {
	clients := make([]*RpcClient, len(aggregators))
	if e := parallel.For(0, len(aggregators), 1, func(i int) error {
		a, e := rpc.DialHTTP("tcp", aggregators[i])
		if e != nil {
			return fmt.Errorf("dial aggregator %s: %v", aggregators[i], e)
		}
		clients[i] = &RpcClient{a, aggregators[i]}
		return nil
	}); e != nil {
		return nil, e
	}
	return clients, nil
}

func closeAll(closers []*RpcClient) error {
	return parallel.For(0, len(closers), 1, func(i int) error {
		return closers[i].Close()
	})
}
