package gibbs

import (
	"github.com/wangkuiyi/phoenix/core/hist"
	"reflect"
	"testing"
)

func TestOptimizerCollectDocumentStatistics(t *testing.T) {
	v, _ := CreateTestingVocabulary()
	d := CreateTestingDocument(v)
	o := NewOptimizer(testingK)
	o.CollectDocumentStatistics(d)
	o.CollectDocumentStatistics(d)

	testingOptimizer := &Optimizer{
		docLenHist: hist.Sparse{2: 2},
		topicDocHists: []hist.Sparse{
			hist.Sparse{},
			hist.Sparse{2: 2}}}
	if !reflect.DeepEqual(o, testingOptimizer) {
		t.Errorf("Expecting o = %v, Got %v", *testingOptimizer, *o)
	}
}

func TestOptimizerOptimize(t *testing.T) {
	m, _, e := CreateTestingOptimizedModel()
	if e != nil {
		t.Skip(e)
	}

	testingLearnedModelAndPrior := Model{
		GlobalTopicHist: hist.Dense{4, 4},
		WordTopicHists: []hist.Hist{
			hist.Sparse{1: 2},
			hist.Sparse{0: 2},
			hist.Sparse{1: 2},
			hist.Sparse{0: 2}},
		TopicPrior:    []float64{0.0234919576942991, 0.023385212706843114},
		TopicPriorSum: 0.04687717040114221,
		WordPrior:     0.01,
		WordPriorSum:  0.04}

	if !reflect.DeepEqual(*m, testingLearnedModelAndPrior) {
		t.Errorf("Expecting model %v. Got %v",
			testingLearnedModelAndPrior, *m)
	}
}
