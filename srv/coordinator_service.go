package srv

import (
	"expvar"
	"fmt"
	"github.com/wangkuiyi/parallel"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Coordinator maintains a group of data server instances and
// a group of sampling server instances.
type Coordinator struct {
	cfg   *Config
	me    string
	squad *Squad

	loaders  []*RpcClient
	samplers []*RpcClient

	// Write to this channel to notify func main() to exit.
	done chan bool

	// mutex protects loaders and samplers.
	mutex sync.Mutex
}

// RunCoordinator creates and run a coordinator RPC service.
func RunCoordinator(cfg *Config, addr string) error {
	sid := cfg.SquadId(addr)
	if sid < 0 {
		return fmt.Errorf("addr %s not in config %+v", addr, *cfg)
	}

	c := &Coordinator{
		cfg:      cfg,
		me:       addr,
		squad:    &cfg.Squads[sid],
		loaders:  make([]*RpcClient, 0, cfg.NumVShards),
		samplers: make([]*RpcClient, 0, cfg.NumVShards),
		done:     make(chan bool),
	}
	rpc.Register(c)
	rpc.HandleHTTP()

	expvar.Publish("config", c.cfg)
	expvar.Publish("samplers",
		expvar.Func(func() interface{} { return c.samplers }))
	expvar.Publish("loaders",
		expvar.Func(func() interface{} { return c.loaders }))

	l, e := net.Listen("tcp", addr)
	if e != nil {
		return fmt.Errorf("listen on %s: %v", addr, e)
	}
	log.Print("Coordinator listen on ", addr)
	go http.Serve(l, nil)

	time.Sleep(1 * time.Second)
	defer func() {
		KillWorkers(c.squad.Samplers)
		KillWorkers(c.squad.Loaders)
	}()
	if e := LaunchWorkers(c.me, "sampler", c.squad.Samplers, cfg); e != nil {
		return e
	}

	sig := make(chan os.Signal, 1) // Signal channel must be buffered.
	signal.Notify(sig, os.Interrupt, os.Kill)
	select {
	case <-c.done:
		log.Printf("Coordinator %s finished. Stopping squad", addr)
	case <-sig:
		return fmt.Errorf("Coordinator %s got SIGKILL/INT", addr)
	}
	return nil
}

// RegisterLoader is expected to be called by loaders to register
// themselves to their coordinator.  RegisterLoader checks if all
// expected loader had registered themselves after each successful
// registeration.  If so, it calls Coordinator.run().
func (c *Coordinator) RegisterLoader(addr string, _ *int) error {
	log.Printf("Received loader registeration from %s\n", addr)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.loaders) >= c.cfg.NumVShards {
		return fmt.Errorf("Register more than enough loader %s", addr)
	}

	if cl, e := rpc.DialHTTP("tcp", addr); e == nil {
		log.Printf("Register loader %s as the %d-th.", addr, len(c.loaders))
		c.loaders = append(c.loaders, &RpcClient{cl, addr})
		if len(c.loaders) >= c.cfg.NumVShards {
			if len(c.cfg.Master) > 0 {
				log.Println("All loaders registered. Start run()")
				go c.run()
			} else {
				log.Println("Master is empty. Consider this a test run.")
			}
		}
	} else {
		return fmt.Errorf("Failed connect loader %s: %v", addr, e)
	}

	return nil
}

// RegisterSampler is expected to be called samplers to register
// themselves to their coordinator.  RegisterSampler checks if all
// expected samplers had registered themselves after each successful
// registration.  If so, it invokes launchLoaders.
func (c *Coordinator) RegisterSampler(addr string, _ *int) error {
	log.Printf("Received sampler registeration from %s\n", addr)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.samplers) >= c.cfg.NumVShards {
		log.Fatalf("Register more than enough samplers %s: %v",
			addr, c.samplers)
	}

	if cl, e := rpc.DialHTTP("tcp", addr); e == nil {
		log.Printf("Register sampler %s as the %d-th.", addr, len(c.samplers))
		c.samplers = append(c.samplers, &RpcClient{cl, addr})
		if len(c.samplers) >= c.cfg.NumVShards {
			go LaunchWorkers(c.me, "loader", c.squad.Loaders, c.cfg)
		}
	} else {
		log.Fatalf("Failed connect sampler %s: %s", addr, e)
	}

	return nil
}

func (c *Coordinator) run() {
	log.Printf("%s starts working", c.me)

	// Dial master and notify the startup of a squad.
	m, e := rpc.DialHTTP("tcp", c.cfg.Master)
	if e != nil {
		log.Fatalf("%s dials master %s: %v", c.me, c.cfg.Master, e)
	}
	var t Task
	if e = m.Call("Master.RegisterSquad", c.me, &t); e != nil {
		log.Fatalf("%s calls Master.RegisterSquad: %v", c.me, e)
	}

	for e = c.do(&t); e == nil; e = c.do(&t) {
		t.Coord = c.me
		if e = m.Call("Master.CompleteTask", &t, &t); e != nil {
			log.Fatalf("%s calls Master.CompleteTask %+v: %v", c.me, t, e)
		}
		// TODO(wyi): in the future, may be Master.CompleteTask can
		// return "NoMoreTask", and we should notify c.done here.
	}
	log.Fatalf("%s do(%+v): %v", c.me, t, e)
}

func (c *Coordinator) do(t *Task) error {
	if len(t.Shards) > c.cfg.NumVShards {
		return fmt.Errorf("Task %+v contains more shards than NumVShards(%d)",
			t, c.cfg.NumVShards)
	}
	switch t.Action {
	case INIT:
		return c.init(t)
	case GIBBS:
		return c.gibbs(t)
	case LOGLL:
		return c.logll(t)
	}
	return fmt.Errorf("Unknown action %d", t.Action)
}

func (c *Coordinator) init(t *Task) error {
	if e := parallel.For(0, len(t.Shards), 1, func(i int) error {
		if e := c.loaders[i].Call("Loader.Init", t.Shards[i], nil); e != nil {
			return fmt.Errorf("Loader %s init %s: %v",
				c.loaders[i].Name, t.Shards[i], e)
		}
		return nil
	}); e != nil {
		return e
	}

	return nil
}

func (sr *Coordinator) gibbs(t *Task) error {
	return fmt.Errorf("Coordinator.gibbs() is under implementation\n")
}

func (sr *Coordinator) logll(t *Task) error {
	return fmt.Errorf("Coordinator.logll() is under implementation\n")
}
