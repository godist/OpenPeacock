package main

import (
	"flag"
	"fmt"
	"github.com/wangkuiyi/phoenix/srv"
	"log"
)

var (
	parent = flag.String("parent", "", "Address of coordinator")
	addr   = flag.String("addr", "", "Address of sampler")
)

func main() {
	cfg := new(srv.Config)
	cfg.RegisterAsFlag()
	flag.Parse()

	log.SetPrefix(fmt.Sprintf("Phoenix.sampler-%s ", *addr))

	if e := srv.RunSampler(cfg, *parent, *addr); e != nil {
		log.Fatal(e)
	}
}
