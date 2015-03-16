// inspect print content of files in specified Phoenix iteration
// directory in human readable format.  It can print either the model,
// the document with current latent variables, or the log-likelihood
// of till the current iteration.  By default, it prints the model in
// the most recent iteration.  To let inspect know which directory it
// is going to print, users are expected to provide the training
// configuration file.  For example, say we are going to inspect the
// model learned in our example training job, we can do:
/*
  $GOPATH/bin/inspect \
  -config=file:$GOPATH/src/github.com/wangkuiyi/phoenix/cmd/master/example.conf
*/
package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"github.com/wangkuiyi/phoenix/srv"
	"log"
	"path"
	"strings"
)

var (
	config    = flag.String("config", "", "The phoenix config file")
	iteration = flag.Int("iteration", -1, "The iteration to inspect")
	content   = flag.String("content", "model", "{doc, model, logll}")
)

func main() {
	flag.Parse()

	cfg, e := srv.LoadConfig(*config)
	if e != nil {
		log.Fatalf("Cannot load config file %s: %v", *config, e)
	}
	if e := cfg.Validate(); e != nil {
		log.Fatalf("Invalid configuration: %v", e)
	}
	log.Println("Done loading config file")

	if maxIter, e := srv.FindMostRecentCompletedIteration(cfg); e != nil {
		log.Fatalf("Cannot find most recent completed iteration in %s: %v",
			cfg.JobDir, e)
	} else {
		if *iteration > maxIter {
			log.Fatalf("iteraton %d larger than the most recent iteration %d",
				*iteration, maxIter)
		} else if *iteration < 0 {
			log.Printf("iteration %d is negative, set to %d",
				*iteration, maxIter)
			*iteration = maxIter
		}
	}
	dir := path.Join(cfg.JobDir, fmt.Sprintf("%05d", *iteration))

	v := gibbs.NewVocabulary()
	if f, e := file.Open(cfg.VocabFile); e != nil {
		log.Fatalf("Cannot open vocab %s: %v", cfg.VocabFile, e)
	} else if e := v.Load(f); e != nil {
		log.Fatalf("Cannot load vocab %s: %v", cfg.VocabFile, e)
	}

	switch *content {
	case "doc":
		e = dumpDoc(dir)
	case "model":
		e = dumpModel(dir, v)
	case "logll":
		e = dumpLogll(dir)
	default:
		e = fmt.Errorf("Unknown content %s", *content)
	}
	if e != nil {
		log.Fatal(e)
	}
}

func dumpModel(dir string, v *gibbs.Vocabulary) error {
	fis, e := file.List(dir)
	if e != nil {
		return fmt.Errorf("Cannot list %s: %v", dir, e)
	}

	for _, fi := range fis {
		if strings.HasPrefix(fi.Name, srv.MODEL_FILE) {
			mf := path.Join(dir, fi.Name)
			f, e := file.Open(mf)
			if e != nil {
				return fmt.Errorf("Cannot open model file %s: %v", mf, e)
			}

			var m gibbs.Model
			if e := gob.NewDecoder(f).Decode(&m); e != nil {
				f.Close()
				return fmt.Errorf("Failed decoding model from %s: %v", mf, e)
			}
			prettyPrintModel(&m, v)

			f.Close()
		}
	}

	return nil
}

// prettyPrintModel prints model in human readable format.
func prettyPrintModel(m *gibbs.Model, v *gibbs.Vocabulary) {
	for w, h := range m.WordTopicHists {
		if h != nil {
			fmt.Printf("%-10s ", v.Token(int32(w)))
			sh := make([]int, m.GlobalTopicHist.Len())
			h.ForEach(func(t int, c int64) error {
				sh[t] += int(c)
				return nil
			})
			for _, c := range sh {
				fmt.Printf("% 5d ", c)
			}
			fmt.Println()
		}
	}

	fmt.Printf("%-10s ", "<global>")
	m.GlobalTopicHist.ForEach(func(_ int, c int64) error {
		fmt.Printf("% 5d ", c)
		return nil
	})
	fmt.Println()
}

func dumpDoc(dir string) error {
	return fmt.Errorf("dumpDoc is under implementation")
}

func dumpLogll(dir string) error {
	return fmt.Errorf("dumpLogll is under implementation")
}
