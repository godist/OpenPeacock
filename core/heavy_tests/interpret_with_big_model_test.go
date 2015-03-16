package heavy_tests

import (
	"flag"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"github.com/wangkuiyi/phoenix/core/hist"
	"github.com/wangkuiyi/phoenix/core/utils"
	"os"
	"path"
	"testing"
)

var (
	flagRunLongTests = flag.Bool("run_long_tests", false,
		"If to run tests that take a long time to finish")
)

func TestInterpretWithRealModel(t *testing.T) {
	if !*flagRunLongTests {
		t.Skipf("Skip TestInterpretWithRealModel without -run_long_tests")
	}

	v := utils.LoadVocabOrDie(path.Join(os.Getenv("GOPATH"), kBigData, kVocab))
	m := utils.LoadModelOrDie(path.Join(os.Getenv("GOPATH"), kBigData, kModel))
	intr := gibbs.NewInterpreter(m, v, 100)

	for topic := 0; topic < m.NumTopics(); topic++ {
		h := m.GetTopWords(topic)
		if h != nil && h.Len() >= 2 {
			hh := h.(*hist.OrderedSparse)
			words := []string{v.Token(hh.Topics[0]), v.Token(hh.Topics[1])}
			if dist, e := intr.Interpret(words, 50, 100); e != nil {
				t.Errorf("Interpreter.Interpret: %v", e)
			} else if len(dist) <= 0 {
				t.Errorf("Interpret result SparseDist is empty")
			} else if dist[0].Topic != int32(topic) {
				t.Errorf("Expecting %d, got %d", topic, dist[0].Topic)
			}
		}
	}
}
