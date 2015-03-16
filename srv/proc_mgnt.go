package srv

import (
	"fmt"
	"github.com/wangkuiyi/parallel"
	"github.com/wangkuiyi/prism"
	"log"
)

func LaunchSquads(cfg *Config) error {
	log.Println("Try killing squads before launching them ...")
	KillSquads(cfg) // in case there are some left there.

	f, e := cfg.Encode()
	if e != nil {
		return fmt.Errorf("Encoding config %s: %v", cfg, e)
	}

	for _, s := range cfg.Squads {
		e = prism.Launch(s.Coordinator, cfg.DeployDir, "coordinator",
			[]string{"-config=" + f, "-addr=" + s.Coordinator},
			cfg.LogDir, cfg.Retry)
		if e != nil {
			return fmt.Errorf("Launch coordinator %s: %v", s.Coordinator, e)
		}
	}
	return nil
}

func KillSquads(cfg *Config) error {
	for i, _ := range cfg.Squads {
		if e := prism.Kill(cfg.Squads[i].Coordinator); e != nil {
			return fmt.Errorf("Killing %s: %v", cfg.Squads[i].Coordinator, e)
		}
	}
	return nil
}

// KillWorkers tell Prism to kill processes who are listening on
// addrs.  It is used to kill aggregators, or samplers and loaders in
// a squad.
func KillWorkers(addrs []string) error {
	return parallel.For(0, len(addrs), 1, func(i int) error {
		return prism.Kill(addrs[i])
	})
}

// LaunchWorkers launches either samplers or loaders, as specified by
// what, and make them listen on addrs.
func LaunchWorkers(who, what string, addrs []string, cfg *Config) error {
	log.Println("Try killing " + what + " before start them ...")
	KillWorkers(addrs)

	f, e := cfg.Encode()
	if e != nil {
		return fmt.Errorf("%s encode config %v: %v", who, cfg, e)
	}

	return parallel.For(0, len(addrs), 1, func(i int) error {
		return prism.Launch(addrs[i], cfg.DeployDir, what,
			[]string{"-config=" + f, "-addr=" + addrs[i], "-parent=" + who},
			cfg.LogDir, 1)
	})
}
