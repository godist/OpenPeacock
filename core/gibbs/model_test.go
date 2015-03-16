package gibbs

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/wangkuiyi/phoenix/core/hist"
	"reflect"
	"testing"
)

func TestNewModel(t *testing.T) {
	m := CreateTestingModel()
	if m.NumTopics() != testingK {
		t.Errorf("Expecting m.NumTopics %d, got %d", testingK, m.NumTopics())
	}
	if m.VocabSize() != testingV {
		t.Errorf("Expecting m.VocabSize %d, got %d", testingV, m.VocabSize())
	}
}

func TestModelAccumulate(t *testing.T) {
	m := CreateTestingModel()
	s := map[int]hist.Hist{
		0: hist.Sparse{0: 10, 1: 10},
		1: hist.Sparse{1: 20}}
	m.Accumulate(s)

	truth := []hist.Hist{
		hist.Sparse{0: 10, 1: 10},
		hist.Sparse{1: 21},
		nil,
		hist.Sparse{1: 1}}
	if !reflect.DeepEqual(m.WordTopicHists, truth) {
		t.Errorf("Expecting %s, got %s", truth, fmt.Sprint(m.WordTopicHists))
	}
}

func TestModelGobEncoding(t *testing.T) {
	m := CreateTestingModel()
	var b bytes.Buffer
	if e := gob.NewEncoder(&b).Encode(m); e != nil {
		t.Errorf("Cannot gob encoding %v: %v", m, e)
	}
}

func TestCloneByGob(t *testing.T) {
	m := CreateTestingModel()
	c := m.Clone()
	if !reflect.DeepEqual(c, m) {
		t.Errorf("The cloned model does not equal to the original one.")
	}
}

func TestSaveAndLoadModel(t *testing.T) {
	m := CreateTestingModel()
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if e := enc.Encode(m); e != nil {
		t.Errorf("Cannot encoding model: %v", e)
	}

	r := NewModel(testingK, testingV, testingAlpha, testingBeta)
	dec := gob.NewDecoder(&buf)
	if e := dec.Decode(r); e != nil {
		t.Errorf("Cannot decode model: %v", e)
	}

	if !reflect.DeepEqual(m, r) {
		t.Errorf("Loaded model does not equal to original one")
	}
}

func TestPrintTopics(t *testing.T) {
	v, _ := CreateTestingVocabulary()
	m := CreateTestingModel()
	var buf bytes.Buffer
	m.PrintTopics(&buf, v)
	u := "Topic 00000 Nt 00000:\nTopic 00001 Nt 00002: orange (1) apple (1)\n"
	if s := buf.String(); s != u {
		t.Errorf("Expecting\n%s\ngot\n%s", u, s)
	}
}

func TestGetTopWords(t *testing.T) {
	m := CreateTestingModel()
	if o := m.GetTopWords(0); o != nil {
		t.Errorf("Expecting nil hist, got %v", o)
	}
	truth := hist.NewOrderedSparse().Assign(hist.Sparse{1: 1, 3: 1})
	if o := m.GetTopWords(1); !reflect.DeepEqual(o, truth) {
		t.Errorf("Expecting %v, got %v", truth, o)
	}
}

func TestGetTopNWords(t *testing.T) {
	m := CreateTestingModel()
	if o := m.GetTopNWords(0, 0.5); o != nil {
		t.Errorf("Expecting nil hist, got %v", o)
	}
	truth := hist.NewOrderedSparse().Assign(hist.Sparse{1: 1})
	if o := m.GetTopNWords(1, 0.5); !reflect.DeepEqual(o, truth) {
		t.Errorf("Expecting %v, got %v", truth, o)
	}
}
