package hist

import (
	"encoding/gob"
	"fmt"
	"math"
)

// Sparse represents histogram using Go map.  Sparse represents word
// topic-histograms in Phoenix model.
type Sparse map[int32]int32

func init() {
	gob.Register(Sparse{})
}

func NewSparse() Sparse {
	return make(Sparse)
}

func (s Sparse) Clear() {
	for k := range s {
		delete(s, k)
	}
}

func (s Sparse) AssignOrdered(o *OrderedSparse) Sparse {
	s.Clear()
	for i := 0; i < o.Len(); i++ {
		s[o.Topics[i]] = o.Counts[i]
	}
	return s
}

func (s Sparse) Add(o Sparse) {
	for k, v := range o {
		s[k] += v
	}
}

func (s Sparse) Equal(o Sparse) bool {
	if len(s) != len(o) {
		return false
	}
	for k, v := range s {
		if v2, ok := o[k]; !ok || v2 != v {
			return false
		}
	}
	return true
}

func (s Sparse) Len() int {
	return len(s)
}

func (s Sparse) At(topic int) int64 {
	return int64(s[int32(topic)])
}

func (s Sparse) Inc(topic, count int) {
	if count <= 0 {
		panic(fmt.Sprintf("Inc(topic=%d, count=%d): count must > 0",
			topic, count))
	}
	if count > int(math.MaxInt32) {
		panic(fmt.Sprintf("count (%d) larger than MaxInt32", count))
	}
	t := int32(topic)
	if s[t] >= math.MaxInt32-int32(count) {
		panic(fmt.Sprintf("d[%d] = %d overflow", topic, s[t]))
	}
	s[t] += int32(count)
}

func (s Sparse) Dec(topic, count int) {
	if count <= 0 {
		panic(fmt.Sprintf("Dec(topic=%d, count=%d): count must > 0",
			topic, count))
	}
	t := int32(topic)
	s[t] -= int32(count)
	if s[t] == 0 {
		delete(s, t)
	}
}

func (s Sparse) ForEach(p func(topic int, count int64) error) error {
	for i, v := range s {
		if e := p(int(i), int64(v)); e != nil {
			return e
		}
	}
	return nil
}

func (s Sparse) Clone() Hist {
	n := NewSparse()
	for k, v := range s {
		n[k] = v
	}
	return n
}
