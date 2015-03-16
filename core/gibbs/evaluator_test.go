package gibbs

import (
	"fmt"
	"testing"
)

func TestEvaluatorPerplexity(t *testing.T) {
	v, _ := CreateTestingVocabulary()
	d := CreateTestingDocument(v)
	m := CreateTestingModel()
	ev := NewEvaluator(m, 0, nil)
	truth := "-1.4515175322974125 2"
	if s := fmt.Sprint(ev.Perplexity(d)); s != truth {
		t.Errorf("Expecting %s, got %s", truth, s)
	}
}

func TestEvaluatorPerplexityAccel(t *testing.T) {
	v, _ := CreateTestingVocabulary()
	d := CreateTestingDocument(v)
	m := CreateTestingModel()
	s := NewSampler(m)
	ev := NewEvaluator(m, 0, s)
	truth := "-1.4501436605355273 2"
	if s := fmt.Sprint(ev.Perplexity(d)); s != truth {
		t.Errorf("Expecting %s, got %s", truth, s)
	}
}
