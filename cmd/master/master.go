package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/parallel"
	"github.com/wangkuiyi/phoenix/srv"
	"github.com/wangkuiyi/prism"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/rpc"
	"os"
	"os/signal"
	"path"
	"reflect"
	"strings"
)

var (
	cfgFlag = flag.String("config_file", "", "The configuration file name")
)

func main() {
	log.SetPrefix("Phoenix.master ")
	flag.Parse()

	log.Printf("Loading config file %s", *cfgFlag)
	cfg, e := srv.LoadConfig(*cfgFlag)
	if e != nil {
		log.Fatalf("Failed loading config file %s: %v", *cfgFlag, e)
	}

	if e := deploy(cfg); e != nil {
		log.Fatalf("Deploy failed: %v", e)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	done := make(chan bool, 1)
	go serve(cfg, done)

	e = srv.LaunchWorkers(cfg.Master, "aggregator", cfg.Aggregators, cfg)
	if e != nil {
		srv.KillWorkers(cfg.Aggregators)
		log.Fatalf("Failed start aggregators: %v", e)
	}

	select {
	case <-done:
	case <-sig:
	}
	srv.KillSquads(cfg)
	srv.KillWorkers(cfg.Aggregators)
}

func serve(cfg *srv.Config, done chan bool) {
	s, e := srv.NewMaster(cfg, done)
	if e != nil {
		log.Printf("NewMaster failed: %v", e)
		done <- true
		return
	}
	rpc.Register(s)
	rpc.HandleHTTP()

	l, e := net.Listen("tcp", cfg.Master)
	if e != nil {
		log.Printf("Master cannot listen on %s: %v", cfg.Master, e)
		done <- true
		return
	}

	log.Printf("Master listening on %s", cfg.Master)
	if e := http.Serve(l, nil); e != nil {
		log.Printf("Master listening on %s failed: %v", cfg.Master, e)
		done <- true
		return
	}
}

func deploy(cfg *srv.Config) error {
	buildDir := file.LocalPrefix + path.Dir(os.Args[0])
	pub := path.Join(cfg.JobDir, "phoenix-"+cfg.JobName+".zip")
	log.Printf("Publish %s to %s", buildDir, pub)
	if e := prism.Publish(buildDir, pub); e != nil {
		return fmt.Errorf("Publish %s to %s: %v", buildDir, pub, e)
	}

	hosts := make(map[string]int)
	host := func(addr string) string { return strings.Split(addr, ":")[0] }
	hosts[host(cfg.Master)]++
	for i, _ := range cfg.Squads {
		hosts[host(cfg.Squads[i].Coordinator)] = 1
		for j, _ := range cfg.Squads[i].Loaders {
			hosts[host(cfg.Squads[i].Loaders[j])]++
			hosts[host(cfg.Squads[i].Samplers[j])]++
		}
	}

	log.Printf("Deploy to %+v", hosts)
	return parallel.RangeMap(hosts, func(k, _ reflect.Value) error {
		h := k.String()
		if len(h) > 0 {
			if e := prism.Deploy(h, pub, cfg.DeployDir); e != nil {
				return fmt.Errorf("Deploy %s: %v", h, e)
			}
		}
		return nil
	})
}
