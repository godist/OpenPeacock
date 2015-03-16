package heavy_tests

import (
	"bytes"
	"fmt"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"math/rand"
	"strings"
	"testing"
)

var (
	groundTruthDocLen = 21
	groundTruthNumDoc = 500
	groundTruthModel  = [][]float64{
		{1, 1, 1, 0, 0, 0, 0, 0, 0}, // 0* : 0.1
		{0, 0, 0, 1, 1, 1, 0, 0, 0}, // 1* : 0.2
		{0, 0, 0, 0, 0, 0, 1, 1, 1}, // 2* : 0.3
		{1, 0, 0, 1, 0, 0, 1, 0, 0}, // *0 : 0.4
		{0, 1, 0, 0, 1, 0, 0, 1, 0}, // *1 : 0.5
		{0, 0, 1, 0, 0, 1, 0, 0, 1}, // *2 : 0.6
	}
	groundTruthVocab = "00\n01\n02\n10\n11\n12\n20\n21\n22"
	groundTruthAlpha = []float64{0.6, 0.2, 0.3, 0.4, 0.5, 0.6}
	groundTruthBeta  = 2
	groundTruthK     = len(groundTruthAlpha)
	groundTruthV     = len(groundTruthModel[0])

	expectedModel = `Topic 0: 20 (548) 21 (534) 22 (495) 00 (53)
Topic 1: 00 (747) 01 (738) 02 (716)
Topic 2: 10 (524) 00 (510) 20 (458) 01 (50)
Topic 3: 02 (862) 22 (828) 12 (721)
Topic 4: 21 (593) 01 (587) 11 (515) 12 (3)
Topic 5: 11 (408) 12 (310) 10 (300)
`
)

// In their paper "Finding Scientific Topics" on PNAS 2004, Thomas
// Griffith and Mark Steyvers presents a visual method to verify the
// convergence of learning latent Dirichlet allocation models.  This
// method samples synthetic training data from a ground-truth model,
// whose each P(w|z) distribution over a vocabulary with V*V tokens is
// represented by a V*V size of image, where a colored pixel i
// represents that P(w=i|z)=1/V/V.  So the ground-truth model with K
// latent topics consists of K images.  Then, the method learns a
// model of K latent topics from the synthetic training data.
// Similarly, each latent topic can be drawn as an image.  If these
// learned images look similar to those in the ground-truth model,
// then the test passes.
//
// In this program, we extend Griffith's method to consider an
// asynmmetric Dirichlet prior \alpha and a symmetric Dirichlet prior
// \beta.
func testGriffith(t *testing.T) {
	// TODO(wyi): Currently this test always fails. We need to find
	// out why.
	v := gibbs.NewVocabulary()
	e := v.Load(strings.NewReader(groundTruthVocab))
	if e != nil {
		t.Errorf("Cannot load vocab: %v", e)
	}

	rng := rand.New(rand.NewSource(-1))
	corpus := createGriffithTrainingData(v, rng)

	m := gibbs.NewModel(groundTruthK, groundTruthV, kAlpha, kBeta)
	for _, d := range corpus {
		d.ApplyToModel(m)
	}

	s := gibbs.NewSampler(m)
	o := gibbs.NewOptimizer(groundTruthK)
	for iter := 0; iter < 300; iter++ {
		for _, d := range corpus {
			s.Sample(d, rng)
			o.CollectDocumentStatistics(d)
		}
		o.OptimizeTopicPriors(m, kShape, kScale, kOptimIter)
		s.AfterOptimization()
	}

	w := new(bytes.Buffer)
	m.PrintTopics(w, v)
	// BUG(wyi) TODO(wyi): The following test fails.  Maybe due to bugs in
	// func (m *Model) PrintTopics(w io.Writer, v *Vocabulary).
	if w.String() != expectedModel {
		t.Errorf("Expecting %v, got %v", expectedModel, w.String())
	}
}

// createGriffithTrainingData samples synthetic training data from
func createGriffithTrainingData(v *gibbs.Vocabulary,
	rng *rand.Rand) []*gibbs.Document {

	t := make([]int, groundTruthK)
	d := make([]string, groundTruthDocLen)
	r := make([]*gibbs.Document, groundTruthNumDoc)
	for i := 0; i < groundTruthNumDoc; i++ {
		sampleTopicHist(rng, t)
		synthesizeDocument(t, rng, d)
		r[i] = gibbs.InitializeDocument(d, v, groundTruthK, rng)
	}
	return r
}

func synthesizeDocument(topicHist []int, rng *rand.Rand, doc []string) {
	doc = doc[0:0]
	for t, c := range topicHist {
		for i := 0; i < c; i++ {
			doc = append(doc, word(sampleDiscrete(groundTruthModel[t], rng)))
		}
	}
}

func sampleTopicHist(rng *rand.Rand, hist []int) {
	dist := make([]float64, groundTruthK)
	copy(dist, groundTruthAlpha)
	for i, _ := range hist {
		hist[i] = 0
	}
	for i := 0; i < groundTruthDocLen; i++ {
		t := sampleDiscrete(dist, rng)
		dist[t] += 1.0
		hist[t]++
	}
}

func sampleDiscrete(dist []float64, rng *rand.Rand) int {
	if len(dist) <= 0 {
		panic("sample from empty distribution")
	}
	sum := 0.0
	for _, v := range dist {
		if v < 0 {
			panic(fmt.Sprintf("bad dist: %v", dist))
		}
		sum += v
	}
	u := rng.Float64() * sum
	sum = 0
	for i, v := range dist {
		sum += v
		if u < sum {
			return i
		}
	}
	panic("sampleDiscrete gets out of all possiblilities")
}

func word(sample int) string {
	return fmt.Sprintf("%d%d", sample/3, sample%3)
}
