// multithread is a multi-threading command line trainer.
// Usage:
/*
  $GOPATH/bin/multithread \
    -vocab=../singlethread/testdata/vocab \
    -corpus=../singlethread/testdata/corpus \
    -topics=2
*/

package main

import (
	"flag"
	"github.com/wangkuiyi/parallel"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"github.com/wangkuiyi/phoenix/core/utils"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"sync"
)

func main() {
	flagAddr := flag.String("addr", ":6060", "HTTP status page address")
	flagVocab := flag.String("vocab", "./testdata/vocab", "Vocabulary file")
	flagCorpus := flag.String("corpus", "./testdata/corpus", "Corpus file")
	flagMinDocLen := flag.Int("minlen", 1, "minimum document length")
	flagMaxDocLen := flag.Int("maxlen", -1, "maximum document length")
	flagTopics := flag.Int("topics", 10, "Number of topics to be learned")
	flagGibbsIter := flag.Int("gibbs_iter", 100, "Gibbs sampling iterations")
	flagAlpha := flag.Float64("alpha", 0.01, "Topic prior")
	flagBeta := flag.Float64("beta", 0.01, "Word prior")
	flagShape := flag.Float64("shape", 0.0, "Shape")
	flagScale := flag.Float64("scale", 1e7, "Scale")
	flagOptimIter := flag.Int("optim_iter", 10, "Iterations of optimization")
	flagShards := flag.Int("shards", 2, "Number of parallel shards")
	flagGoMaxProcs := flag.Int("GOMAXPROCS", -1, "GOMAXPROCS")
	flagOptimStart := flag.Int("optim_start", 10,
		"The Gibbs sampling iteration since when it optimize hyperparams")
	flagModel := flag.String("model", "", "The model output")
	flagCache := flag.Int("cache", 0, "Smoothing model cache in MB")
	flagEvalLag := flag.Int("eval_lag", 1, "Evaluation lag")
	flag.Parse()

	is := utils.EnableExpvar(*flagAddr)
	log.Printf("Initialization start at %s", is.Start().StartTime)

	// A hack on setting the MAXPROCS.
	if *flagGoMaxProcs < 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(*flagGoMaxProcs)
	}
	log.Println("Running with MAXPROCS ", runtime.GOMAXPROCS(-1))

	vocab := utils.LoadVocabOrDie(*flagVocab)
	rng := rand.New(rand.NewSource(-1))
	corpus := utils.LoadCorpusOrDie(*flagCorpus, vocab, *flagTopics,
		*flagMinDocLen, *flagMaxDocLen, rng)
	model := utils.InitializeModel(corpus, vocab, *flagTopics,
		*flagAlpha, *flagBeta)

	shards := *flagShards
	if shards > len(corpus) {
		shards = len(corpus)
	}

	log.Printf("Initialization done in %s", is.End(0.0).Duration)

	sigs := make(chan os.Signal, 1)
	exit := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		for sig := range sigs {
			log.Printf("Caught signal, will checkpoint and exit ...")
			exit <- sig
		}
	}()

GibbsIterations:
	for iter := 0; iter < *flagGibbsIter; iter++ {
		select {
		case <-exit:
			log.Printf("Early terminated by signal.")
			break GibbsIterations
		default:
		}

		log.Printf("Iteration %04d start at %s", iter, is.Start().StartTime)

		// Create diffs, each record the opinion of a shard.
		diffs := make([]*gibbs.Model, shards)
		for i, _ := range diffs {
			diffs[i] =
				gibbs.NewModel(*flagTopics, vocab.Len(), *flagAlpha, *flagBeta)
		}

		// Parallel Gibbs sampling.
		parallel.For(0, shards, 1, func(i int) error {
			local_model := model.Clone()
			sampler := gibbs.NewSampler(local_model)
			sampler.SetDiff(diffs[i])
			if iter > *flagOptimStart {
				sampler.AfterOptimization()
			}
			rng := rand.New(rand.NewSource(-1))
			for d := i; d < len(corpus); d += shards {
				sampler.Sample(corpus[d], rng)
			}
			return nil
		})

		// Aggregate opinions from all shards.
		for _, diff := range diffs {
			model.Accumulate(
				gibbs.NewSharder(1).ShardModel(diff.WordTopicHists)[0])
		}

		// Hyperparam optimization
		if iter > *flagOptimStart {
			optimizer := gibbs.NewOptimizer(*flagTopics)
			for _, d := range corpus {
				optimizer.CollectDocumentStatistics(d)
			}
			optimizer.OptimizeTopicPriors(model, *flagShape, *flagScale,
				*flagOptimIter)
		}

		// Parallel calculation of log-likelihood.
		if iter%*flagEvalLag == 0 {
			logLL := 0.0
			nw := 0
			// Here we make use of Sampler.buildSmoothingOnlyBucket to
			// accelerate the initialization of Evaluator.
			s := gibbs.NewSampler(model)
			eval := gibbs.NewEvaluator(model, *flagCache, s)
			var muxAdd sync.Mutex
			parallel.For(0, shards, 1, func(i int) error {
				localLogLL := 0.0
				localNW := 0
				for d := i; d < len(corpus); d += shards {
					ll, nw := eval.Perplexity(corpus[d])
					localLogLL += ll
					localNW += nw
				}
				muxAdd.Lock()
				defer muxAdd.Unlock()
				logLL += localLogLL
				nw += localNW
				return nil
			})
			pp := math.Exp(-logLL / float64(nw))
			log.Printf("Iteration %04d perplexity %f", iter, pp)
			log.Printf("Iteration %04d done in %s", iter, is.End(pp).Duration)
		} else {
			log.Printf("Iteration %04d done in %s", iter, is.End(0.0).Duration)
		}
	}

	utils.SaveModel(model, *flagModel)
}
