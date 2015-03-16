package gibbs

import (
	"fmt"
	"github.com/wangkuiyi/phoenix/core/hist"
	"reflect"
	"testing"
)

func TestShardModel(t *testing.T) {
	hists := make([]hist.Hist, 3)
	hists[0] = hist.Sparse{0: 0}
	hists[1] = nil
	hists[2] = hist.Sparse{0: 0, 1: 1}

	groundTruth := []map[int]hist.Hist{
		map[int]hist.Hist{
			0: hist.Sparse{0: 0},
		},
		map[int]hist.Hist{
			2: hist.Sparse{0: 0, 1: 1},
		},
	}
	m := NewSharder(2).ShardModel(hists)
	if !reflect.DeepEqual(m, groundTruth) {
		t.Errorf("Expecting %v, got %v", groundTruth, fmt.Sprint(m))
	}
}
