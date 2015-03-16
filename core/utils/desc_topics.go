package utils

import (
	"fmt"
	"github.com/wangkuiyi/parallel"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"html/template"
	"log"
	"runtime"
)

func DescribeTopics(m *gibbs.Model, v *gibbs.Vocabulary,
	maxWordsPerTopic int) []*TopicDesc {

	log.Printf("Generating topic descriptions ... ")
	descs := make([]*TopicDesc, m.NumTopics())

	parallel.ForN(0, m.NumTopics(), 1, 2*runtime.NumCPU(), func(topic int) {
		h := m.GetTopWords(topic)
		if h == nil {
			panic(fmt.Sprintf("topic %d got empty word list", topic))
		}
		descs[topic] = &TopicDesc{
			Id:     topic,
			Nt:     m.GlobalTopicHist.At(topic),
			Tokens: make([]TokenDesc, 0, maxWordsPerTopic)}
		i := 0
		h.ForEach(func(t int, count int64) error {
			if i < maxWordsPerTopic {
				descs[topic].Tokens = append(descs[topic].Tokens,
					TokenDesc{template.HTML(v.Token(int32(t))), count})
			}
			i++
			return nil
		})
	})

	log.Printf("Done generating topic descriptions.")
	return descs
}

type TopicDesc struct {
	Id     int
	Nt     int64
	Tokens []TokenDesc
}
type TokenDesc struct {
	Word  template.HTML
	Count int64
}
