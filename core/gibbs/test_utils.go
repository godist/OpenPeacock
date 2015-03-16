package gibbs

import (
	"fmt"
	"math/rand"
	"strings"
)

const (
	testingV = 4

	testingAlpha = 0.1
	testingBeta  = 0.01
	testingK     = 2

	testingShape           = 0.0
	testingScale           = 1e7
	testingOptimIter       = 5
	testingTotalIterations = 110
)

func CreateTestingDocument(v *Vocabulary) *Document {
	rng := rand.New(rand.NewSource(1))
	return InitializeDocument([]string{"apple", "unknown", "orange"},
		v, testingK, rng)
}

// CreateTestingVocabulary creates a model with testingV tokens
func CreateTestingVocabulary() (*Vocabulary, error) {
	r := strings.NewReader("apple 100\norange	whatever\n\ncat\ntiger")
	v := NewVocabulary()
	e := v.Load(r)
	return v, e
}

// CreateTestingModel creates a model with:
//  symmetric topic prior: 0.1
//  symmetric word prior:  0.01
//  word states:   topic 0    topic 1
//        tiger:   <nil>
//       orange:   nil        1
//          cat:   <nil>
//        apple:   nil        1
//  global states: topic 0    topic 1
//                 0          2
func CreateTestingModel() *Model {
	v, e := CreateTestingVocabulary()
	if e != nil {
		panic("CreateTestingModel failed at CreateTestingVocabulary")
	}
	d := CreateTestingDocument(v)
	m := NewModel(testingK, testingV, testingAlpha, testingBeta)
	d.ApplyToModel(m)
	return m
}

// CreateTestingOptimizedModel learns a model with hyper-parameter
// optimization.
func CreateTestingOptimizedModel() (*Model, *Vocabulary, error) {
	v, e := CreateTestingVocabulary()
	if e != nil {
		return nil, nil, fmt.Errorf("Failed building testing vocabulary")
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
	o := NewOptimizer(testingK)
	for iter := 0; iter < testingTotalIterations; iter++ {
		for _, d := range corpus {
			s.Sample(d, rng)
			o.CollectDocumentStatistics(d)
		}
		o.OptimizeTopicPriors(m, testingShape, testingScale, testingOptimIter)
		s.buildSmoothingOnlyBucket()
		s.cacheCoefficients()
	}

	return m, v, nil
}
