package utils

import (
	cmprs "github.com/wangkuiyi/compress_io"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
)

func TestLoadVocabOrDie(t *testing.T) {
	dir, e := ioutil.TempDir("", "")
	if e != nil {
		t.Fatalf("Cannot create temp dir: %v", e)
	}
	defer os.RemoveAll(dir)

	v, e := gibbs.CreateTestingVocabulary()
	if e != nil {
		t.Fatalf("CreateTestingVocabulary: %v", e)
	}

	gzFile := createTempVocab(dir, ".gz", strings.Join(v.Tokens, "\n"))
	if len(gzFile) == 0 {
		t.Fatalf("createTempVocab failed")
	}
	defer os.Remove(gzFile)

	v2 := LoadVocabOrDie(gzFile)
	if !reflect.DeepEqual(v, v2) {
		t.Errorf("Expecting\n%v\ngot\n%v\n", v, v2)
	}

	plainFile := createTempVocab(dir, "", strings.Join(v.Tokens, "\n"))
	if len(plainFile) == 0 {
		t.Fatalf("createTempVocab failed")
	}
	defer os.Remove(plainFile)

	v2 = LoadVocabOrDie(plainFile)
	if !reflect.DeepEqual(v, v2) {
		t.Errorf("Expecting\n%v\ngot\n%v\n", v, v2)
	}

}

func TestLoadTranslationOrDie(t *testing.T) {
	dir, e := ioutil.TempDir("", "")
	if e != nil {
		t.Fatalf("Cannot create temp dir: %v", e)
	}
	defer os.RemoveAll(dir)

	v, e := gibbs.CreateTestingVocabulary()
	if e != nil {
		t.Fatalf("CreateTestingVocabulary: %v", e)
	}

	gzFile := createTempVocab(dir, ".gz", strings.Join(v.Tokens, "\n"))
	if len(gzFile) == 0 {
		t.Fatalf("createTempVocab failed")
	}
	defer os.Remove(gzFile)

	trans := make([]string, len(v.Tokens))
	truth := make([]string, len(v.Tokens))
	for i, tok := range v.Tokens {
		trans[i] = tok + " " + "The " + tok
		truth[i] = "The " + tok
	}
	transFile := createTempFile(dir, "trans", ".gz", strings.Join(trans, "\n"))
	if len(transFile) == 0 {
		t.Fatalf("createTempFile failed")
	}
	defer os.Remove(transFile)

	v = LoadVocabOrDie(gzFile)
	tr := LoadTranslationOrDie(transFile)
	v1 := TranslatedVocab(v, tr)
	if !reflect.DeepEqual(v1.Tokens, truth) {
		t.Errorf("Expecting\n%v\ngot\n%v\n", truth, v.Tokens)
	}
}

func TestLoadCorpusOrDie(t *testing.T) {
	dir, e := ioutil.TempDir("", "")
	if e != nil {
		t.Fatalf("Cannot create temp dir: %v", e)
	}
	defer os.RemoveAll(dir)

	// Following content is copied from and must match core/gibbs/test_utils.go
	content := "apple unknown orange\n"
	rng := rand.New(rand.NewSource(1))

	v, e := gibbs.CreateTestingVocabulary()
	if e != nil {
		t.Fatalf("CreateTestingVocabulary: %v", e)
	}
	d := gibbs.CreateTestingDocument(v)

	plainFile := createTempCorpus(dir, "", content)
	if len(plainFile) == 0 {
		t.Fatalf("createTempCorpus failed")
	}

	c := LoadCorpusOrDie(plainFile, v, 2, 1, 50, rng)
	if !reflect.DeepEqual(c[0].Words, d.Words) {
		t.Errorf("Expecting %v, got %v", c[0].Words, d.Words)
	}

	gzFile := createTempCorpus(dir, ".gz", content)
	if len(gzFile) == 0 {
		t.Fatalf("createTempCorpus failed")
	}

	if !reflect.DeepEqual(c[0].Words, d.Words) {
		t.Errorf("Expecting %v, got %v", c[0].Words, d.Words)
	}

}

func TestSaveAndLoadModelOrDie(t *testing.T) {
	dir, e := ioutil.TempDir("", "")
	if e != nil {
		t.Fatalf("Cannot create temp dir: %v", e)
	}
	defer os.RemoveAll(dir)

	m := gibbs.CreateTestingModel()

	gzFile := path.Join(dir, "model.gz")
	SaveModel(m, gzFile)
	m1 := LoadModelOrDie(gzFile)
	if !reflect.DeepEqual(*m, *m1) {
		t.Errorf("Expecting\n%v\ngot\n%v\n", *m, *m1)
	}

	plainFile := path.Join(dir, "model")
	SaveModel(m, plainFile)
	m1 = LoadModelOrDie(plainFile)
	if !reflect.DeepEqual(*m, *m1) {
		t.Errorf("Expecting\n%v\ngot\n%v\n", *m, *m1)
	}
}

func createTempVocab(dir, ext, content string) string {
	return createTempFile(dir, "vocab", ext, content)
}

func createTempCorpus(dir, ext, content string) string {
	return createTempFile(dir, "corpus", ext, content)
}

func createTempFile(dir, name, ext, content string) string {
	filename := path.Join(dir, name+ext)
	f, e := os.Create(filename)
	w := cmprs.NewWriter(f, e, path.Ext(filename))
	if w == nil {
		log.Printf("NewCompressWriter failed")
		return ""
	}
	defer w.Close()

	if _, e := w.Write([]byte(content)); e != nil {
		log.Printf("Failed writing to temp file %s: %v", filename, e)
	}

	return filename
}
