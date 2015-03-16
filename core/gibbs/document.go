package gibbs

import (
	"github.com/wangkuiyi/phoenix/core/hist"
	"math/rand"
)

type Document struct {
	TopicHist *hist.OrderedSparse
	Words     []int32
	Topics    []int32
}

func (d *Document) Len() int {
	return len(d.Words)
}

func InitializeDocument(words []string, vocab *Vocabulary, numTopics int,
	rng *rand.Rand) *Document {
	d := &Document{
		Words:     make([]int32, 0, len(words)),
		Topics:    make([]int32, 0, len(words)),
		TopicHist: hist.NewOrderedSparseAndReserve(len(words)),
	}
	for i := range words {
		if id := vocab.Id(words[i]); id >= 0 {
			d.Words = append(d.Words, id)
			topic := rng.Intn(numTopics)
			d.Topics = append(d.Topics, int32(topic))
			d.TopicHist.Inc(topic, 1)
		}
	}
	return d
}

func (d *Document) ApplyToModel(m *Model) {
	for i := range d.Words {
		m.WordTopicHist(d.Words[i]).Inc(int(d.Topics[i]), 1)
		m.GlobalTopicHist.Inc(int(d.Topics[i]), 1)
	}
}
