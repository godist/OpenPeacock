package hist

import (
	"encoding/gob"
	"fmt"
	"math"
)

// Dense is a plain histogram represented by a count array. It can be
// used to represent the global topic histogram of Phoenix.
type Dense []int64

func init() {
	gob.Register(Dense{})
}

func NewDense(dim int) Dense {
	return make(Dense, int(dim), int(dim))
}

func (d Dense) At(topic int) int64 {
	return d[topic]
}

func (d Dense) Inc(topic, count int) {
	if count < 0 {
		panic(fmt.Sprintf("count (%d) is negative", count))
	}
	if d[topic] >= math.MaxInt64-int64(count) {
		panic(fmt.Sprintf("d[%d] = %d overflow", topic, d[topic]))
	}
	d[topic] += int64(count)
}

func (d Dense) Dec(topic, count int) {
	if count < 0 {
		panic(fmt.Sprintf("count (%d) is negative", count))
	}
	d[topic] -= int64(count)
}

func (d Dense) Len() int {
	return len(d)
}

func (d Dense) ForEach(p func(topic int, count int64) error) error {
	for i, v := range d {
		if e := p(i, int64(v)); e != nil {
			return e
		}
	}
	return nil
}

func (d Dense) Clone() Hist {
	n := NewDense(d.Len())
	copy(n, d)
	return n
}
