package srv

import (
	"expvar"
	_ "expvar"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/rpc"
)

type Sampler struct {
	cfg         *Config
	coord       string
	me          string
	squad       *Squad
	aggregators []*RpcClient
	done        chan bool
}

func RunSampler(cfg *Config, coord, sampler string) error {
	cid, sid := cfg.SamplerId(coord, sampler)
	if cid < 0 || sid < 0 {
		return fmt.Errorf("Cannot identify coord (%s) or sampler (%s)",
			coord, sampler)
	}

	as, e := connectToAggregators(cfg.Aggregators)
	if e != nil {
		if len(cfg.Aggregators) > 0 {
			return fmt.Errorf("Connect to %v: %v", cfg.Aggregators, e)
		} else {
			log.Print("Aggregators is empty. Consider this a test run.")
			as = nil
		}
	}

	s := &Sampler{
		cfg:         cfg,
		coord:       coord,
		me:          sampler,
		squad:       &cfg.Squads[cid],
		aggregators: as,
		done:        make(chan bool),
	}
	rpc.Register(s)
	rpc.HandleHTTP()

	expvar.Publish("config", s.cfg)
	expvar.Publish("coord", expvar.Func(func() interface{} { return s.coord }))
	expvar.Publish("me", expvar.Func(func() interface{} { return s.me }))
	expvar.Publish("aggregators",
		expvar.Func(func() interface{} { return s.aggregators }))

	l, e := net.Listen("tcp", sampler)
	if e != nil {
		log.Fatalf("listen on %s error: %v", sampler, e)
	}
	log.Printf("Sampler started by %s listen on %s", coord, sampler)
	go http.Serve(l, nil)

	if e := registerSampler(cfg, coord, sampler); e != nil {
		return fmt.Errorf("Cannot register sampler %s: %v", sampler, e)
	}

	<-s.done
	return nil
}

func registerSampler(cfg *Config, coord, sampler string) error {
	cl, e := rpc.DialHTTP("tcp", coord)
	if e != nil {
		return fmt.Errorf("Failed dialing %s: %v", coord, e)
	}

	e = cl.Call("Coordinator.RegisterSampler", sampler, nil)
	if e != nil {
		return fmt.Errorf("Failed register %s: %v", sampler, e)
	}

	return nil
}

func (s *Sampler) Sample(_ int, _ *int) error { return nil }
