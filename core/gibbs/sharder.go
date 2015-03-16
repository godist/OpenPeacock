package gibbs

import (
	"fmt"
	"github.com/wangkuiyi/phoenix/core/hist"
)

// Sharder defines a sequence of fixed number of buckets, and the
// allocation of a zero-based sequence of integers into these buckets.
// The allocations follows the principle that these buckets have
// similar size.
type Sharder struct {
	Shards int
}

func NewSharder(shards int) Sharder {
	if shards <= 0 {
		panic(fmt.Sprintf("shards (%d) <= 0", shards))
	}
	return Sharder{shards}
}

// ShardModel divides a slice of histograms into the number of buckets
// defined by NewSharder.  Each bucket is represented by a map from
// index to histogram.
func (s Sharder) ShardModel(hists []hist.Hist) []map[int]hist.Hist {
	m := make([]map[int]hist.Hist, s.Shards)
	v := len(hists)
	b := s.Shards
	if v < b {
		b = v
	}
	bucketSize := v / b
	extendedBucketSize := bucketSize + 1
	extendedBuckets := v % b
	normalBuckets := b - extendedBuckets

	for j := 0; j < extendedBuckets; j++ {
		m[j] = make(map[int]hist.Hist)
		for k := 0; k < extendedBucketSize; k++ {
			t := j*extendedBucketSize + k
			if h := hists[t]; h != nil {
				m[j][t] = h
			}
		}
	}

	for j := 0; j < normalBuckets; j++ {
		m[j+extendedBuckets] = make(map[int]hist.Hist)
		for k := 0; k < bucketSize; k++ {
			t := extendedBuckets*extendedBucketSize + j*bucketSize + k
			h := hists[t]
			if h != nil {
				m[j+extendedBuckets][t] = h
			}
		}
	}

	return m
}
