package gibbs

import (
	"fmt"
	"github.com/wangkuiyi/phoenix/core/hist"
	"math/rand"
	"reflect"
	"testing"
)

const (
	testingSmoothingOnlyBucketFactors = "[0.025 0.0004901960784313725]"
	testingSmoothingOnlyBucketSize    = "0.0254902"
	testingDocumentTopicBucketFactors = "[0 0.00980392156862745]"
	testingDocumentTopicBucketSize    = "0.00980392156862745"
	testingTopicWordBucketFactors     = "[0 0.049019607843137254]"
	testingTopicWordBucketSize        = "0.049019607843137254"
	testingCachedCoefficients         = "[2.5 0.049019607843137254]"
	testingUpdatedCoefficients        = "[2.5 1.0294117647058825]"
)

var (
	sprint = fmt.Sprint // A shortcut to fmt.Sprint
)

func TestSamplerBuildSmoothingOnlyBucket(t *testing.T) {
	m := CreateTestingModel()
	s := NewSampler(m)
	if sprint(s.smoothingOnlyBucketFactors) !=
		testingSmoothingOnlyBucketFactors {
		t.Errorf("Expecting s.smoothingOnlyBuketFactors = %s. Got %s",
			testingSmoothingOnlyBucketFactors,
			sprint(s.smoothingOnlyBucketFactors))
	}
	if fmt.Sprintf("%.7f", s.smoothingOnlyBucketSize) !=
		testingSmoothingOnlyBucketSize {
		t.Errorf("Expecting s.smoothingOnlyBucketSize = %s. Got %.7f",
			testingSmoothingOnlyBucketSize, s.smoothingOnlyBucketSize)
	}
}

func TestSamplerBuildDocumentTopicBucket(t *testing.T) {
	m := CreateTestingModel()
	s := NewSampler(m)
	if s.documentTopicBucketSize != 0 {
		t.Errorf("Expecting documentTopicBucketSize = 0. Got %f",
			s.documentTopicBucketSize)
	}
	if len(s.documentTopicBucketFactors) != testingK {
		t.Errorf("Expecting len(s.documentTopicBucketFactors) = %d. Got %d",
			testingK, len(s.documentTopicBucketFactors))
	}

	v, e := CreateTestingVocabulary()
	if e != nil {
		t.Errorf("Failed building testing vocabulary")
	}
	d := CreateTestingDocument(v)
	s.buildDocumentTopicBucket(d)
	if sprint(s.documentTopicBucketFactors) !=
		testingDocumentTopicBucketFactors {
		t.Errorf("Expecting s.documentTopicBucketFactors = %s. Got %v",
			testingDocumentTopicBucketFactors, s.documentTopicBucketFactors)
	}
	if sprint(s.documentTopicBucketSize) != testingDocumentTopicBucketSize {
		t.Errorf("Expecting s.smoothingOnlyBucketSize = %s. Got %.7f",
			testingDocumentTopicBucketSize, s.documentTopicBucketSize)
	}
}

func TestSamplerBuildTopicWordBucket(t *testing.T) {
	m := CreateTestingModel()
	s := NewSampler(m)
	if s.topicWordBucketSize != 0 {
		t.Errorf("Expecting s.topicWordBucketSize = 0. Got %f",
			s.topicWordBucketSize)
	}
	if len(s.topicWordBucketFactors) != testingK {
		t.Errorf("Expecting len(s.topicWordBucketFactors) = %d. Got %d",
			testingK, len(s.topicWordBucketFactors))
	}

	s.cacheCoefficients()            // depends on coefficients.
	s.buildTopicWordBucket(int32(1)) // token id of "orange"
	if sprint(s.topicWordBucketFactors) != testingTopicWordBucketFactors {
		t.Errorf("Expecting s.topicWordBucketFactors = %s. Got %v",
			testingTopicWordBucketFactors, s.topicWordBucketFactors)
	}
	if sprint(s.topicWordBucketSize) != testingTopicWordBucketSize {
		t.Errorf("Expecting s.topicWordBucketSize = %s. Got %s",
			testingTopicWordBucketSize, sprint(s.topicWordBucketSize))
	}
}

func TestSamplerCacheUpdateResetCoefficients(t *testing.T) {
	m := CreateTestingModel()
	s := NewSampler(m)
	if sprint(s.coefficients) != testingCachedCoefficients {
		t.Errorf("Expecting s.coefficients = %s. Got %s",
			testingCachedCoefficients, sprint(s.coefficients))
	}

	v, e := CreateTestingVocabulary()
	if e != nil {
		t.Errorf("Failed building testing vocabulary")
	}
	d := CreateTestingDocument(v)
	s.updateCoefficients(d)
	if sprint(s.coefficients) != testingUpdatedCoefficients {
		t.Errorf("Expecting s.coefficients = %s. Got %s",
			testingUpdatedCoefficients, sprint(s.coefficients))
	}

	s.resetCoefficients(d)
	if sprint(s.coefficients) != testingCachedCoefficients {
		t.Errorf("Expecting s.coefficients = %s. Got %s",
			testingCachedCoefficients, sprint(s.coefficients))
	}
}

func TestSamplerSample(t *testing.T) {
	v, e := CreateTestingVocabulary()
	if e != nil {
		t.Errorf("Failed building testing vocabulary")
	}

	rng := rand.New(rand.NewSource(-1))

	corpus := []*Document{
		InitializeDocument([]string{"apple", "orange"}, v, testingK, rng),
		InitializeDocument([]string{"orange", "apple"}, v, testingK, rng),
		InitializeDocument([]string{"cat", "tiger"}, v, testingK, rng),
		InitializeDocument([]string{"tiger", "cat"}, v, testingK, rng),
	}

	m := NewModel(testingK, testingV, testingAlpha, testingBeta)
	for _, d := range corpus {
		d.ApplyToModel(m)
	}

	s := NewSampler(m)
	for iter := 0; iter < testingTotalIterations; iter++ {
		for _, d := range corpus {
			s.Sample(d, rng)
		}
	}

	testingLearnedModel := &Model{
		GlobalTopicHist: hist.Dense{4, 4},
		WordTopicHists: []hist.Hist{
			hist.Sparse{1: 2},
			hist.Sparse{0: 2},
			hist.Sparse{1: 2},
			hist.Sparse{0: 2}},
		TopicPrior:    []float64{0.1, 0.1},
		TopicPriorSum: 0.2,
		WordPrior:     0.01,
		WordPriorSum:  0.04}
	if !reflect.DeepEqual(m, testingLearnedModel) {
		t.Errorf("Expecting %v. Got %v", testingLearnedModel, m)
	}
}

func TestSamplerDiff(t *testing.T) {
	v, e := CreateTestingVocabulary()
	if e != nil {
		t.Errorf("Failed building testing vocabulary")
	}

	rng := rand.New(rand.NewSource(-1))

	corpus := []*Document{
		InitializeDocument([]string{"apple", "orange"}, v, testingK, rng),
		InitializeDocument([]string{"orange", "apple"}, v, testingK, rng),
		InitializeDocument([]string{"cat", "tiger"}, v, testingK, rng),
		InitializeDocument([]string{"tiger", "cat"}, v, testingK, rng),
	}

	m := NewModel(testingK, testingV, testingAlpha, testingBeta)
	d := NewModel(testingK, testingV, testingAlpha, testingBeta)
	for _, doc := range corpus {
		doc.ApplyToModel(m)
		doc.ApplyToModel(d)
	}

	s := NewSampler(m)
	s.SetDiff(d)
	for iter := 0; iter < testingTotalIterations; iter++ {
		for _, d := range corpus {
			s.Sample(d, rng)
		}
	}

	if sprint(*m) != sprint(*d) {
		t.Errorf("model does not equal to diff. Model:\n%v\nDiff:\n%v", *m, *d)
	}
}
