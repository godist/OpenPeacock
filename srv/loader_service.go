package srv

import (
	"bufio"
	"encoding/gob"
	"expvar"
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/parallel"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"hash/fnv"
	"log"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/rpc"
	"path"
	"strings"
)

type Loader struct {
	cfg      *Config
	coord    string
	me       string
	squad    *Squad
	samplers []*RpcClient
	done     chan bool
}

// RunLoader creates and runs a Loader RPC service.
func RunLoader(cfg *Config, coord, loader string) error {
	sid, lid := cfg.LoaderId(coord, loader)
	if sid < 0 || lid < 0 {
		return fmt.Errorf("Cannot identify coord (%s) or loader (%s)",
			coord, loader)
	}

	ss, e := connectToSamplers(cfg.Squads[sid].Samplers)
	if e != nil {
		return fmt.Errorf("Cannot connect to samplers: %v", e)
	}

	s := &Loader{
		cfg:      cfg,
		coord:    coord,
		me:       loader,
		squad:    &cfg.Squads[sid],
		samplers: ss,
		done:     make(chan bool),
	}
	rpc.Register(s)
	rpc.HandleHTTP()

	expvar.Publish("config", s.cfg)
	expvar.Publish("coord", expvar.Func(func() interface{} { return s.coord }))
	expvar.Publish("me", expvar.Func(func() interface{} { return s.me }))
	expvar.Publish("samplers",
		expvar.Func(func() interface{} { return s.samplers }))

	l, e := net.Listen("tcp", loader)
	if e != nil {
		return fmt.Errorf("listen on %s error: %v", loader, e)
	}
	log.Printf("Loader started by %s listen on %s", coord, loader)
	go http.Serve(l, nil)

	if e := registerLoader(cfg, coord, loader); e != nil {
		return fmt.Errorf("Cannot register loader %s: %v", loader, e)
	}

	<-s.done
	return nil
}

func registerLoader(cfg *Config, coord, loader string) error {
	cl, e := rpc.DialHTTP("tcp", coord)
	if e != nil {
		return fmt.Errorf("Failed dialing %s: %v", coord, e)
	}

	e = cl.Call("Coordinator.RegisterLoader", loader, nil)
	if e != nil {
		return fmt.Errorf("Failed register %s: %v", loader, nil)
	}

	return nil
}

// Init accepts shard, the basename of an input shard in the directory
// of CorpusDir.
func (l *Loader) Init(shard string, _ *int) error {
	// Open the input shard file.
	me := l.cfg.Squads[0].Loaders[0]
	in, e := file.Open(path.Join(l.cfg.CorpusDir, shard))
	if e != nil {
		return fmt.Errorf("%s open shard %s: %v", me, shard, e)
	}
	defer in.Close()

	// Create the output shard file.
	oshard := path.Join(l.cfg.JobDir, fmt.Sprintf("%05d", 0), shard)
	o, e := file.Create(oshard)
	if e != nil {
		return fmt.Errorf("%s create shard %s: %v", me, oshard, e)
	}
	b := bufio.NewWriter(o)
	defer func() {
		b.Flush()
		o.Close()
	}()

	// Load vocabulary and creating an empty model.
	v := gibbs.NewVocabulary()
	if f, e := file.Open(l.cfg.VocabFile); e != nil {
		return fmt.Errorf("%s open vocab file %s: %v", me, l.cfg.VocabFile, e)
	} else if e := v.Load(f); e != nil {
		return fmt.Errorf("Load vocab %s: %v", l.cfg.VocabFile, e)
	}
	m := gibbs.NewModel(l.cfg.NumTopics, v.Len(), l.cfg.TopicPrior,
		l.cfg.WordPrior)

	// Load the initialize shard files.
	hasher := fnv.New64a()
	hasher.Write([]byte(shard))
	rng := rand.New(rand.NewSource(int64(hasher.Sum64())))
	s := bufio.NewScanner(in)
	en := gob.NewEncoder(b)
	for s.Scan() {
		words := strings.Split(s.Text(), " ")
		d := gibbs.InitializeDocument(words, v, l.cfg.NumTopics, rng)
		d.ApplyToModel(m)
		if e := en.Encode(d); e != nil {
			return fmt.Errorf("%s encode document %+v: %v", me, d, e)
		}
	}
	if e := s.Err(); e != nil {
		return fmt.Errorf("%s scans shard %s: %v", me, shard, e)
	}

	// Connect to aggregators, and close these connections before return.
	aggregators, e := connectToAggregators(l.cfg.Aggregators)
	if e != nil {
		return fmt.Errorf("loader %s cannot connect to aggregators", e)
	}
	defer closeAll(aggregators)

	// Shard local model matrix and send them to aggregators.
	numShards := len(aggregators)
	shards := gibbs.NewSharder(numShards).ShardModel(m.WordTopicHists)
	if e := parallel.For(0, numShards, 1, func(i int) error {
		e := aggregators[i].Call("Aggregator.Init", shards[i], nil)
		if e != nil {
			return fmt.Errorf("failed to call %s", aggregators[i].Name)
		}
		return nil
	}); e != nil {
		return fmt.Errorf("Loader %s %s", me, e)
	}

	return nil
}
