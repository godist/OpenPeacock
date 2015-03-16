// singlethread is a single-threaded command line trainer.
// Usage:
/*
  $GOPATH/bin/singlethread \
    -vocab=./testdata/vocab -corpus=./testdata/corpus -topics=2
*/

package main

import (
	"flag"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"github.com/wangkuiyi/phoenix/core/utils"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
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
	flagOptimStart := flag.Int("optim_start", 10,
		"The Gibbs sampling iteration since when it optimize hyperparams")
	flagShape := flag.Float64("shape", 0.0, "Shape")
	flagScale := flag.Float64("scale", 1e7, "Scale")
	flagOptimIter := flag.Int("optim_iter", 10, "Iterations of optimization")
	flagModel := flag.String("model", "", "The model output")
	flagCache := flag.Int("cache", 0, "Smoothing model cache in MB")
	flagEvalLag := flag.Int("eval_lag", 1, "Evaluation lag")
	flag.Parse()

	is := utils.EnableExpvar(*flagAddr)
	log.Printf("Initialization start at %s", is.Start().StartTime)

	vocab := utils.LoadVocabOrDie(*flagVocab)
	rng := rand.New(rand.NewSource(-1))
	corpus := utils.LoadCorpusOrDie(*flagCorpus, vocab, *flagTopics,
		*flagMinDocLen, *flagMaxDocLen, rng)
	model := utils.InitializeModel(corpus, vocab, *flagTopics,
		*flagAlpha, *flagBeta)
	sampler := gibbs.NewSampler(model)

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

		optimizer := gibbs.NewOptimizer(*flagTopics)
		for _, d := range corpus {
			sampler.Sample(d, rng)
			if iter > *flagOptimStart {
				optimizer.CollectDocumentStatistics(d)
			}
		}

		if iter > *flagOptimStart {
			optimizer.OptimizeTopicPriors(model, *flagShape, *flagScale,
				*flagOptimIter)
			sampler.AfterOptimization()
		}

		if iter%*flagEvalLag == 0 {
			eval := gibbs.NewEvaluator(model, *flagCache, sampler)
			logL := 0.0
			nW := 0
			for d := 0; d < len(corpus); d++ {
				ll, nw := eval.Perplexity(corpus[d])
				logL += ll
				nW += nw
			}
			pp := math.Exp(-logL / float64(nW))
			log.Printf("Iteration %04d perplexity %f", iter, pp)
			log.Printf("Iteration %04d done in %s", iter, is.End(pp).Duration)
		} else {
			log.Printf("Iteration %04d done in %s", iter, is.End(0.0).Duration)
		}
	}

	utils.SaveModel(model, *flagModel)
}
