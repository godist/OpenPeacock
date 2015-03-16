package srv

import (
	"encoding/gob"
	"expvar"
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"github.com/wangkuiyi/phoenix/core/hist"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"path"
)

type Aggregator struct {
	cfg   *Config
	me    string
	done  chan bool
	vocab *gibbs.Vocabulary
	model *gibbs.Model
}

func RunAggregator(cfg *Config, addr string) error {
	if aid := cfg.AggregatorId(addr); aid < 0 {
		return fmt.Errorf("Aggregator %s not in config %+v", addr, cfg)
	}

	v, e := loadVocabulary(cfg)
	if e != nil {
		return e
	}
	m := gibbs.NewModel(cfg.NumTopics, v.Len(), cfg.TopicPrior, cfg.WordPrior)

	s := &Aggregator{
		cfg:   cfg,
		me:    addr,
		done:  make(chan bool, 1),
		vocab: v,
		model: m,
	}
	rpc.Register(s)
	rpc.HandleHTTP()

	expvar.Publish("config", s.cfg)
	expvar.Publish("me", expvar.Func(func() interface{} { return s.me }))

	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatalf("listen on %s error: %v", addr, e)
	}
	go http.Serve(l, nil)

	log.Println("Aggregator listen on ", addr)
	if e := registerAggregator(cfg, addr); e != nil {
		if len(cfg.Master) > 0 {
			return e
		}
		log.Print("cfg.Master is empty. Consider this a test run.")
	}

	<-s.done
	return nil
}

func loadVocabulary(cfg *Config) (*gibbs.Vocabulary, error) {
	vf, e := file.Open(cfg.VocabFile)
	if e != nil {
		return nil, e
	}
	defer vf.Close()

	v := gibbs.NewVocabulary()
	if e := v.Load(vf); e != nil {
		return nil, e
	}
	return v, nil
}

func registerAggregator(cfg *Config, me string) error {
	mr, e := rpc.DialHTTP("tcp", cfg.Master)
	if e != nil {
		return fmt.Errorf("Failed dialing %s: %v", cfg.Master, e)
	}

	e = mr.Call("Master.RegisterAggregator", me, nil)
	if e != nil {
		return fmt.Errorf("Failed register aggregator %s: %v", me, nil)
	}

	return nil
}

func (s *Aggregator) Init(hists map[int]hist.Hist, _ *int) error {
	s.model.Accumulate(hists)
	return nil
}

func (s *Aggregator) Save(is *struct{ Iter, VShard, VShards int },
	_ *int) error {
	p := path.Join(s.cfg.JobDir, fmt.Sprintf("%05d", is.Iter),
		fmt.Sprintf("%s-%05d-of-%05d", MODEL_FILE, is.VShard, is.VShards))
	f, e := file.Create(p)
	if e != nil {
		return fmt.Errorf("Cannot create file %s: %v", p, e)
	}
	defer f.Close()
	if e := gob.NewEncoder(f).Encode(s.model); e != nil {
		return fmt.Errorf("Failed encoding to %s: %v", p, e)
	}
	return nil
}

func (a *Aggregator) GetGlobalHist(_ *int, ret *hist.Dense) error {
	*ret = a.model.GlobalTopicHist.(hist.Dense)
	return nil
}

func (a *Aggregator) SetGlobalHist(gh hist.Dense, _ *int) error {
	a.model.GlobalTopicHist = gh
	return nil
}
