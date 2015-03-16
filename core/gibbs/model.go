package gibbs

import (
	"fmt"
	"github.com/wangkuiyi/phoenix/core/hist"
	"io"
)

type Model struct {
	GlobalTopicHist hist.Hist
	WordTopicHists  []hist.Hist
	TopicPrior      []float64
	TopicPriorSum   float64
	WordPrior       float64
	WordPriorSum    float64
}

func NewModel(numTopics, vocabSize int, topicPrior, wordPrior float64) *Model {
	if numTopics < 2 {
		panic(fmt.Sprintf("numTopics = %d, less than 2", numTopics))
	}
	if vocabSize < 2 {
		panic(fmt.Sprintf("vocabSize = %d, less than 2", vocabSize))
	}
	if topicPrior <= 0.0 {
		panic(fmt.Sprintf("topicPrior = %f, less than 0", topicPrior))
	}
	if wordPrior <= 0.0 {
		panic(fmt.Sprintf("wordPrior = %f, less than 0", wordPrior))
	}
	m := &Model{
		GlobalTopicHist: hist.NewDense(numTopics),
		WordTopicHists:  make([]hist.Hist, vocabSize),
		TopicPrior:      make([]float64, numTopics),
		TopicPriorSum:   topicPrior * float64(numTopics),
		WordPrior:       wordPrior,
		WordPriorSum:    wordPrior * float64(vocabSize),
	}
	for i := range m.TopicPrior {
		m.TopicPrior[i] = topicPrior
	}
	return m
}

func (m *Model) NumTopics() int {
	return m.GlobalTopicHist.Len()
}

func (m *Model) VocabSize() int {
	return cap(m.WordTopicHists)
}

func (m *Model) WordTopicHist(token int32) hist.Hist {
	if h := m.WordTopicHists[token]; h != nil {
		return h
	}
	h := hist.NewSparse()
	m.WordTopicHists[token] = h
	return h
}

func (m *Model) PrintTopics(w io.Writer, v *Vocabulary) {
	m.PrintTopicsTopNWords(w, v, 1.0)
}

// PrintTopicsTopNWords prints each topic as words with P(w|z) in
// descending order.  Parameter percentage is used to call
// Model.GetTopNWords() and controls how many words to be printed for
// each topic.
func (m *Model) PrintTopicsTopNWords(w io.Writer, v *Vocabulary,
	percentage float64) {

	m.GlobalTopicHist.ForEach(func(topic int, count int64) error {
		fmt.Fprintf(w, "Topic %05d Nt %05d:", topic, count)
		if h := m.GetTopNWords(topic, percentage); h != nil {
			h.ForEach(func(t int, count int64) error {
				fmt.Fprintf(w, " %s (%d)", v.Token(int32(t)), count)
				return nil
			})
		}
		fmt.Fprintf(w, "\n")
		return nil
	})
}

// GetTopWords returns tokens in a given topic and their weights.
func (m *Model) GetTopWords(topic int) hist.Hist {
	wordHist := hist.NewSparse()
	for word, h := range m.WordTopicHists {
		if h != nil {
			h.ForEach(func(t int, c int64) error {
				if t == topic {
					wordHist.Inc(word, int(c))
				}
				return nil
			})
		}
	}

	if len(wordHist) > 0 {
		return hist.NewOrderedSparse().Assign(wordHist)
	}
	return nil
}

// GetTopNWords returns top-N tokens in a topic where these tokens'
// weight accumulate to percetage of total weight of the topic.
func (m *Model) GetTopNWords(topic int, percentage float64) hist.Hist {
	// GetTopNWords requires the GetTopWords returns hist.OrderedSparse.
	if o := m.GetTopWords(topic); o != nil {
		h := o.(*hist.OrderedSparse)
		var accum int
		for i := 0; i < h.Len(); i++ {
			accum += int(h.Counts[i])
			if accum >= int(float64(m.GlobalTopicHist.At(topic))*percentage) {
				h.Topics = h.Topics[0 : i+1]
				h.Counts = h.Counts[0 : i+1]
				break
			}
		}
		return h
	}
	return nil
}

func (m *Model) Accumulate(hists map[int]hist.Hist) {
	for w, h := range hists {
		if d := m.WordTopicHists[w]; d == nil {
			// Fast copy
			m.WordTopicHists[w] = h
		} else {
			h.ForEach(func(t int, c int64) error {
				if c > 0 {
					d.Inc(t, int(c))
				} else if c < 0 {
					d.Dec(t, int(-c))
				}
				return nil
			})
		}

		h.ForEach(func(t int, c int64) error {
			if c > 0 {
				m.GlobalTopicHist.Inc(t, int(c))
			} else if c < 0 {
				m.GlobalTopicHist.Dec(t, int(-c))
			}
			return nil
		})
	}
}

// Clone uses gob.Encode/Decode to do deep clone of a model.
func (m *Model) Clone() *Model {
	n := NewModel(m.NumTopics(), m.VocabSize(), 1.0, 1.0)

	copy(n.TopicPrior, m.TopicPrior)
	n.TopicPriorSum = m.TopicPriorSum
	n.WordPrior = m.WordPrior
	n.WordPriorSum = m.WordPriorSum
	copy(n.GlobalTopicHist.(hist.Dense), m.GlobalTopicHist.(hist.Dense))
	for w, h := range m.WordTopicHists {
		if h == nil {
			n.WordTopicHists[w] = nil
		} else {
			n.WordTopicHists[w] = h.Clone()
		}
	}

	return n
}
