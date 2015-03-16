package gibbs

import (
	"errors"
	"log"
	"math/rand"
)

// Sampler implements the SparseLDA sampling algorithm as described in
// the paper *Topic Model Inference on Streaming Document Collections*
// by Limin Yao, David Mimno, and Andrew McCallum at KDD in 2009.
// This algorithm supports the use of an asymmetric Dirichlet prior.
type Sampler struct {
	model                      *Model
	diff                       *Model
	smoothingOnlyBucketSize    float64 // equation (7)
	smoothingOnlyBucketFactors []float64
	documentTopicBucketSize    float64 // equation (8)
	documentTopicBucketFactors []float64
	topicWordBucketSize        float64 // equation (9)
	topicWordBucketFactors     []float64
	coefficients               []float64 // part of equation (10)
}

func NewSampler(m *Model) *Sampler {
	s := &Sampler{
		model: m,
		smoothingOnlyBucketSize:    0,
		smoothingOnlyBucketFactors: make([]float64, m.NumTopics()),
		documentTopicBucketSize:    0,
		documentTopicBucketFactors: make([]float64, m.NumTopics()),
		topicWordBucketSize:        0,
		topicWordBucketFactors:     make([]float64, m.NumTopics()),
		coefficients:               make([]float64, m.NumTopics()),
	}
	s.buildSmoothingOnlyBucket()
	s.cacheCoefficients()
	return s
}

// SetDiff let Sampler knowns about an empty model; and
// Sampler.Sample() will record subsequent Gibbs udpates into it.
// This is convenient to implement distributed Gibbs sampling. If you
// pass nil as the parameter, Sampler does not record updates.
func (s *Sampler) SetDiff(d *Model) {
	s.diff = d
}

// GetDiff retrieves a model representing Gibbs updates happend since
// the pevious invocation to SetDiff.
func (s *Sampler) GetDiff() *Model {
	return s.diff
}

func (s *Sampler) AfterOptimization() {
	s.buildSmoothingOnlyBucket()
	s.cacheCoefficients()
}

func (s *Sampler) buildSmoothingOnlyBucket() {
	s.smoothingOnlyBucketSize = 0
	for t := range s.smoothingOnlyBucketFactors {
		s.smoothingOnlyBucketFactors[t] = 0
	}
	for t := 0; t < s.model.NumTopics(); t++ {
		s.smoothingOnlyBucketFactors[t] =
			s.model.TopicPrior[t] * s.model.WordPrior /
				(s.model.WordPriorSum + float64(s.model.GlobalTopicHist.At(t)))
		s.smoothingOnlyBucketSize += s.smoothingOnlyBucketFactors[t]
	}
}

func (s *Sampler) buildDocumentTopicBucket(doc *Document) {
	s.documentTopicBucketSize = 0
	for t := range s.documentTopicBucketFactors {
		s.documentTopicBucketFactors[t] = 0
	}
	for i := 0; i < doc.TopicHist.Len(); i++ {
		t := int(doc.TopicHist.Topics[i])
		s.documentTopicBucketFactors[t] =
			s.model.WordPrior * float64(doc.TopicHist.Counts[i]) /
				(s.model.WordPriorSum + float64(s.model.GlobalTopicHist.At(t)))
		s.documentTopicBucketSize += s.documentTopicBucketFactors[t]
	}
}

// buildTopicWordBucket assumes that cacheCoefficients had been called
// to fill s.coefficients.
func (s *Sampler) buildTopicWordBucket(token int32) {
	s.topicWordBucketSize = 0
	for t := range s.topicWordBucketFactors {
		s.topicWordBucketFactors[t] = 0
	}
	h := s.model.WordTopicHist(token)
	h.ForEach(func(t int, c int64) error {
		s.topicWordBucketFactors[t] = s.coefficients[t] * float64(c)
		s.topicWordBucketSize += s.topicWordBucketFactors[t]
		return nil
	})
}

// cacheCoefficients computes only the smoothing part of equation
// (10), which need to be complemented by the rest parts in the Gibbs
// sampling of a document, by calling updateCoefficients, and then be
// reset to smoothing-only part after processing the document by
// calling resetCoefficients.
func (s *Sampler) cacheCoefficients() {
	for t := 0; t < s.model.NumTopics(); t++ {
		s.coefficients[t] = s.model.TopicPrior[t] /
			(s.model.WordPriorSum + float64(s.model.GlobalTopicHist.At(t)))
	}
}

// updateCoefficients is called by SampleNewTopics as we begin with a
// document.  It only updates coefficents that correspond to topics
// with non-zero histogram count in the document.
func (s *Sampler) updateCoefficients(doc *Document) {
	for i := 0; i < doc.TopicHist.Len(); i++ {
		t := int(doc.TopicHist.Topics[i])
		s.coefficients[t] =
			(s.model.TopicPrior[t] + float64(doc.TopicHist.Counts[i])) /
				(s.model.WordPriorSum + float64(s.model.GlobalTopicHist.At(t)))
	}
}

// resetCoefficients is called after we processeed a document.  It
// resets those coefficients that corresponds to topics with non-zero
// histrogram count in the document.
func (s *Sampler) resetCoefficients(doc *Document) {
	for i := 0; i < doc.TopicHist.Len(); i++ {
		t := int(doc.TopicHist.Topics[i])
		s.coefficients[t] = s.model.TopicPrior[t] /
			(s.model.WordPriorSum + float64(s.model.GlobalTopicHist.At(t)))
	}
}

func (s *Sampler) neglectOrConsiderWord(
	doc *Document, token int32, topic int32, neglect bool) {

	t := int(topic)
	if neglect {
		s.model.WordTopicHist(token).Dec(t, 1)
		s.model.GlobalTopicHist.Dec(t, 1)
		doc.TopicHist.Dec(t, 1)

		if s.diff != nil {
			s.diff.WordTopicHist(token).Dec(t, 1)
			s.diff.GlobalTopicHist.Dec(t, 1)
		}
	} else {
		s.model.WordTopicHist(token).Inc(t, 1)
		s.model.GlobalTopicHist.Inc(t, 1)
		doc.TopicHist.Inc(t, 1)

		if s.diff != nil {
			s.diff.WordTopicHist(token).Inc(t, 1)
			s.diff.GlobalTopicHist.Inc(t, 1)
		}
	}

	s.smoothingOnlyBucketSize -= s.smoothingOnlyBucketFactors[topic]
	s.documentTopicBucketSize -= s.documentTopicBucketFactors[topic]

	docTopicCount := float64(doc.TopicHist.At(t))
	globalTopicCount := float64(s.model.GlobalTopicHist.At(t))

	s.smoothingOnlyBucketFactors[topic] =
		s.model.TopicPrior[topic] * s.model.WordPrior /
			(s.model.WordPriorSum + globalTopicCount)
	s.documentTopicBucketFactors[topic] =
		docTopicCount * s.model.WordPrior /
			(s.model.WordPriorSum + globalTopicCount)

	s.smoothingOnlyBucketSize += s.smoothingOnlyBucketFactors[topic]
	s.documentTopicBucketSize += s.documentTopicBucketFactors[topic]

	s.coefficients[topic] =
		(s.model.TopicPrior[topic] + docTopicCount) /
			(s.model.WordPriorSum + globalTopicCount)
}

func (s *Sampler) sampleNewTopic(doc *Document, token int32,
	rng *rand.Rand) int32 {
	norm := s.smoothingOnlyBucketSize +
		s.documentTopicBucketSize + s.topicWordBucketSize
	draw := rng.Float64() * norm
	var newTopic int32 = -1

	if draw < s.topicWordBucketSize {
		s.model.WordTopicHist(token).ForEach(func(topic int, _ int64) error {
			draw -= s.topicWordBucketFactors[topic]
			if draw <= 0 {
				newTopic = int32(topic)
				return errors.New("break")
			}
			return nil
		})
	} else {
		draw -= s.topicWordBucketSize
		if draw < s.documentTopicBucketSize {
			for i := 0; i < doc.TopicHist.Len(); i++ {
				topic := doc.TopicHist.Topics[i]
				draw -= s.documentTopicBucketFactors[topic]
				if draw <= 0 {
					newTopic = topic
					break
				}
			}
		} else {
			draw -= s.documentTopicBucketSize
			var i int32
			draw -= s.smoothingOnlyBucketFactors[i]
			for draw > 0 {
				i++
				draw -= s.smoothingOnlyBucketFactors[i]
			}
			newTopic = i
		}
	}

	if newTopic < 0 || int(newTopic) >= s.model.NumTopics() {
		log.Fatalf("Failed in sampling: newTopic = %d out of range [0, %d)",
			newTopic, s.model.NumTopics())
	}
	return newTopic
}

func (s *Sampler) Sample(doc *Document, rng *rand.Rand) {
	s.buildDocumentTopicBucket(doc)
	s.updateCoefficients(doc)
	for i := 0; i < doc.Len(); i++ {
		token := doc.Words[i]
		oldTopic := doc.Topics[i]
		s.neglectOrConsiderWord(doc, token, oldTopic, true)
		s.buildTopicWordBucket(token)
		newTopic := s.sampleNewTopic(doc, token, rng)
		doc.Topics[i] = newTopic
		s.neglectOrConsiderWord(doc, token, newTopic, false)
	}
	s.resetCoefficients(doc)
}
