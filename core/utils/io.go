package utils

import (
	"bufio"
	"encoding/gob"
	cmprs "github.com/wangkuiyi/compress_io"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
)

func LoadVocabOrDie(filename string) *gibbs.Vocabulary {
	log.Printf("Loading vocab %s ... ", filename)

	f, e := os.Open(filename)
	r := cmprs.NewReader(f, e, path.Ext(filename))
	if r == nil {
		log.Fatalf("Cannot open vocab file %s: %v", filename, e)
	}

	defer r.Close()
	vocab := gibbs.NewVocabulary()
	if e := vocab.Load(r); e != nil {
		log.Fatalf("Failed loading vocab file %s: %v", filename, e)
	}

	log.Println("Done loading vocabulary.")
	return vocab
}

func LoadCorpusOrDie(filename string, vocab *gibbs.Vocabulary, topics int,
	minLen, maxLen int, rng *rand.Rand) []*gibbs.Document {

	log.Printf("Loading corpus %s ... ", filename)

	f, e := os.Open(filename)
	r := cmprs.NewReader(f, e, path.Ext(filename))
	if r == nil {
		log.Fatalf("Cannot open corpus file %s: %v", filename, e)
	}

	defer r.Close()
	corpus := make([]*gibbs.Document, 0)
	scanned := 0
	s := bufio.NewReader(r)
	for {
		line, e := s.ReadString('\n')
		if e != nil {
			if e != io.EOF {
				log.Fatal("Error reading", filename, ":", e)
			} else {
				break
			}
		}
		scanned++

		tokens := strings.Fields(line)
		d := gibbs.InitializeDocument(tokens, vocab, topics, rng)
		if ((minLen > 0 && d.Len() >= minLen) || minLen <= 0) &&
			((maxLen > 0 && d.Len() <= maxLen) || maxLen <= 0) {
			corpus = append(corpus, d)
		}
	}

	if len(corpus) > 0 {
		log.Printf("Done loading corpus: %d out of %d.", len(corpus), scanned)
	} else {
		log.Fatal("corpus contain no valid document!")
	}
	return corpus
}

func LoadModelOrDie(filename string) *gibbs.Model {
	log.Printf("Loading model %s ...", filename)
	m := new(gibbs.Model)

	f, e := os.Open(filename)
	r := cmprs.NewReader(f, e, path.Ext(filename))
	if r == nil {
		log.Fatalf("Cannot open model file %s: %v", filename, e)
	}
	defer r.Close()

	dec := gob.NewDecoder(r)
	if e := dec.Decode(m); e != nil {
		log.Fatalf("Cannot decode model: %v", e)
	}

	log.Printf("Done. %d topics %d tokens.", m.NumTopics(), m.VocabSize())
	return m
}

func InitializeModel(corpus []*gibbs.Document, vocab *gibbs.Vocabulary,
	topics int, alpha, beta float64) *gibbs.Model {

	log.Print("Initializing model ... ")
	model := gibbs.NewModel(topics, vocab.Len(), alpha, beta)
	for _, d := range corpus {
		d.ApplyToModel(model)
	}
	log.Println("Done initializing model.")
	return model
}

func SaveModel(model *gibbs.Model, filename string) {
	if len(filename) > 0 {
		f, e := os.Create(filename)
		w := cmprs.NewWriter(f, e, path.Ext(filename))
		if w == nil {
			log.Printf("Cannot create file %s: %v", filename, e)
		} else {
			defer func() {
				w.Close()
				log.Printf("Saved model to %s.", filename)
			}()
			enc := gob.NewEncoder(w)
			if e := enc.Encode(model); e != nil {
				log.Printf("Failed encoding model: %v", e)
			}
		}
	}
}

type Trans map[string]string

func TranslatedVocab(v *gibbs.Vocabulary, tr Trans) *gibbs.Vocabulary {
	log.Printf("Translating vocabulary ... ")
	for i, s := range v.Tokens {
		if t, exist := tr[s]; exist {
			v.Tokens[i] = t
		} else {
			log.Printf("Cannot translate %s", s)
		}
	}
	log.Printf("Done with translating vocabulary.")
	return v
}

func LoadTranslationOrDie(filename string) Trans {
	log.Printf("Loading translation %s ...", filename)
	trans := make(map[string]string)

	f, e := os.Open(filename)
	if r := cmprs.NewReader(f, e, path.Ext(filename)); r == nil {
		log.Fatalf("Cannot load from %s", filename)
	} else {
		defer r.Close()
		s := bufio.NewScanner(r)
		for s.Scan() {
			fs := strings.Fields(s.Text())
			if len(fs) < 2 {
				log.Fatalf("%v has less than 2 fields", fs)
			}
			if _, exist := trans[fs[0]]; exist {
				log.Fatalf("Found duplicated company Id (%s) in %s", fs[0], fs)
			}
			trans[fs[0]] = strings.Join(fs[1:len(fs)], " ")
		}
		if e := s.Err(); e != nil {
			log.Fatalf("Reading %s error: %v", filename, e)
		}
	}

	log.Printf("Done loading translation,  %d entries.", len(trans))
	return trans
}
