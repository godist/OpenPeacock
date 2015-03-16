package main

import (
	"flag"
	"fmt"
	"github.com/wangkuiyi/phoenix/srv"
	"log"
)

var (
	addr = flag.String("addr", "", "Address of aggregator")

	// parent is not a must-to-have for aggregator. It is just for
	// interface consistency with sampler and loader, so we can start
	// all of them using launchWorkers.
	parent = flag.String("parent", "", "Address of master")
)

func main() {
	cfg := new(srv.Config)
	cfg.RegisterAsFlag()
	flag.Parse()

	log.SetPrefix(fmt.Sprintf("Phoenix.aggregator-%s ", *addr))

	if e := srv.RunAggregator(cfg, *addr); e != nil {
		log.Fatal(e)
	}
}
