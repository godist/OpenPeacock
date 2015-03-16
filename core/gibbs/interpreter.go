package gibbs

import (
	"errors"
	"fmt"
	"github.com/wangkuiyi/phoenix/core/hist"
	"hash/fnv"
	"math/rand"
	"sort"
	"strings"
)

const (
	ErrEmptyDoc = "Interpret empty document."
)

type Interpreter struct {
	model            *ModelAccessor
	vocab            *Vocabulary
	smoothingOnlySum []float64
}

func NewInterpreter(m *Model, v *Vocabulary, cacheMB int) *Interpreter {
	accessor := NewModelAccessor(m, cacheMB)
	return &Interpreter{
		model:            accessor,
		vocab:            v,
		smoothingOnlySum: computeWordTopicPriorSum(accessor)}
}

func computeWordTopicPriorSum(model *ModelAccessor) []float64 {
	smoothingOnlySum := make([]float64, model.VocabSize())
	for word, _ := range model.WordTopicHists {
		var sum float64
		smoothingOnlyBucket := model.WordTopicDist(int32(word))
		for topic, _ := range smoothingOnlyBucket {
			sum += model.TopicPrior[topic] * smoothingOnlyBucket[topic]
		}
		smoothingOnlySum[word] = sum
	}
	return smoothingOnlySum
}

func (intr *Interpreter) Interpret(words []string, burnin, iter int) (
	SparseDist, error) {
	if iter <= burnin {
		panic(fmt.Sprintf("iter (%d) <= burin (%d)", iter, burnin))
	}

	hasher := fnv.New64()
	hasher.Write([]byte(strings.Join(words, "\t")))
	rng := rand.New(rand.NewSource(int64(hasher.Sum64())))
	doc := InitializeDocument(words, intr.vocab, intr.model.NumTopics(), rng)
	if doc.Len() <= 0 {
		return nil, errors.New(ErrEmptyDoc)
	}
	cache := newDistCache(intr.model)
	accumulatedTopicHist := hist.NewSparse()
	norm := 0.0

	for i := 0; i < iter; i++ {
		for j := 0; j < doc.Len(); j++ {
			word := doc.Words[j]
			oldTopic := doc.Topics[j]
			doc.TopicHist.Dec(int(oldTopic), 1)
			smoothingOnlyBucket := cache.Get(word)
			newTopic := intr.sampleTopic(doc, word, smoothingOnlyBucket, rng)
			doc.Topics[j] = newTopic
			doc.TopicHist.Inc(int(newTopic), 1)
		}

		if i > burnin {
			doc.TopicHist.ForEach(func(topic int, count int64) error {
				accumulatedTopicHist.Inc(topic, int(count))
				norm += float64(count)
				return nil
			})
		}
	}

	dist := make(SparseDist, accumulatedTopicHist.Len())
	i := 0
	accumulatedTopicHist.ForEach(func(topic int, count int64) error {
		dist[i].Topic = int32(topic)
		dist[i].Prob = float64(count) / norm
		i++
		return nil
	})
	sort.Sort(dist)
	return dist, nil
}

type distCache struct {
	accessor *ModelAccessor
	cache    map[int32][]float64
}

func newDistCache(a *ModelAccessor) *distCache {
	return &distCache{
		accessor: a,
		cache:    make(map[int32][]float64)}
}

func (c *distCache) Get(word int32) []float64 {
	if dist, ok := c.cache[word]; ok {
		return dist
	}
	dist := c.accessor.WordTopicDist(word)
	c.cache[word] = dist
	return dist
}

func (intr *Interpreter) sampleTopic(doc *Document, word int32,
	smoothingOnlyBucket []float64, rng *rand.Rand) int32 {

	docTopicBucket, docTopicSum := intr.calculateDocumentTopicBucket(
		doc, word, smoothingOnlyBucket)
	var newTopic int32 = -1
	sample := rng.Float64() * (docTopicSum + intr.smoothingOnlySum[int(word)])

	if sample < docTopicSum { // sample is in document topic bucket
		for i := 0; i < len(docTopicBucket); i++ {
			sample -= docTopicBucket[i].Prob
			if sample <= 0 {
				newTopic = docTopicBucket[i].Topic
				break
			}
		}
	} else { // sample is in smoothing only bucket
		sample -= docTopicSum
		i := 0
		sample -= smoothingOnlyBucket[i] * intr.model.TopicPrior[i]
		for sample > 0 {
			i++
			sample -= smoothingOnlyBucket[i] * intr.model.TopicPrior[i]
		}
		if i >= intr.model.NumTopics() {
			panic(fmt.Sprintf("i (%d) >= model.NumTopics() (%d)",
				i, intr.model.NumTopics()))
		}
		newTopic = int32(i)
	}

	if newTopic < 0 {
		panic(fmt.Sprintf("newTopic (%d) < 0", newTopic))
	}
	return newTopic
}

func (intr *Interpreter) calculateDocumentTopicBucket(doc *Document,
	word int32, smoothingOnlyBucket []float64) (SparseDist, float64) {

	docTopicBucket := make(SparseDist, 0, doc.Len())
	var docTopicSum float64

	doc.TopicHist.ForEach(func(topic int, count int64) error {
		docTopicBucket = append(docTopicBucket, Prob{
			int32(topic),
			float64(count) * smoothingOnlyBucket[topic]})
		docTopicSum += float64(count) * smoothingOnlyBucket[topic]
		return nil
	})
	return docTopicBucket, docTopicSum
}

type Prob struct {
	Topic int32
	Prob  float64
}
type SparseDist []Prob

func (a SparseDist) Len() int           { return len(a) }
func (a SparseDist) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SparseDist) Less(i, j int) bool { return a[i].Prob > a[j].Prob }
