package heavy_tests

import (
	"bytes"
	"encoding/gob"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"github.com/wangkuiyi/phoenix/core/utils"
	"os"
	"path"
	"testing"
)

func cloneModelByGob(m *gibbs.Model) *gibbs.Model {
	var buf bytes.Buffer
	e := gob.NewEncoder(&buf)
	e.Encode(m)
	d := gob.NewDecoder(&buf)
	c := gibbs.NewModel(m.NumTopics(), m.VocabSize(), m.TopicPrior[0],
		m.WordPrior)
	d.Decode(c)
	return c
}

func BenchmarkCloneModelByGob(b *testing.B) {
	m := utils.LoadModelOrDie(path.Join(os.Getenv("GOPATH"), kBigData, kModel))
	for i := 0; i < b.N; i++ {
		cloneModelByGob(m)
	}
}

func BenchmarkCloneModelByHistClone(b *testing.B) {
	m := utils.LoadModelOrDie(path.Join(os.Getenv("GOPATH"), kBigData, kModel))
	for i := 0; i < b.N; i++ {
		m.Clone()
	}
}
