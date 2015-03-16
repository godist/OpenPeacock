// coordinator starts an RPC server and launches samplers and loaders
// in its squad.  Each sampler or loader process is monitors by a
// goroutine.  Whenever a process fails, the monitor goroutine trys to
// restart it, for at most max_retry times.
//
// To run a simple test instance of coordinator with a squad of two
// samplers and two loaders on the local computer, please try the
// following bash command:
package main

import (
	"flag"
	"fmt"
	"github.com/wangkuiyi/phoenix/srv"
	"log"
)

var (
	addr = flag.String("addr", "", "The address of coordinator")
)

func main() {
	cfg := new(srv.Config)
	cfg.RegisterAsFlag()
	flag.Parse()

	log.SetPrefix(fmt.Sprintf("Phoenix.coordinator-%s ", *addr))

	if e := srv.RunCoordinator(cfg, *addr); e != nil {
		log.Fatalf("Failed start squad %s: %v", *addr, e)
	}
}
