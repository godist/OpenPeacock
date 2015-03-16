package hist

import (
	"fmt"
	"math"
	"sort"
)

// OrderedSparse represent a histogram using two arrays, Topics and
// Counts, where Counts is in descending order.  This property can be
// used to accelerate the Gibbs sampling of documents, so it
// represents document topic-histograms.
type OrderedSparse struct {
	Topics []int32
	Counts []int32
}

func NewOrderedSparse() *OrderedSparse {
	return &OrderedSparse{nil, nil}
}

// In some cases, we know the maximum number of non-zeros in the
// histogram.  For example, when we use OrderedSparse as document
// topic histogram, the maximum number of non-zeros is min(numTopics,
// docLength).  In such cases, we can reserve capacity in order to
// reduce the cost of memory re-allocation in increaseCapacity().
func NewOrderedSparseAndReserve(cap int) *OrderedSparse {
	return &OrderedSparse{
		Topics: make([]int32, 0, cap),
		Counts: make([]int32, 0, cap)}
}

// Len makes OrderedSparse compatible with sort.Interface.
func (o *OrderedSparse) Len() int {
	return len(o.Topics)
}

// Less allows package sort to sort elements in OrderedSparse
// descreasing order.
func (o *OrderedSparse) Less(i, j int) bool {
	return o.Counts[i] > o.Counts[j] ||
		(o.Counts[i] == o.Counts[j] &&
			o.Topics[i] < o.Topics[j])
}

// Swap makes OrderedSparse compatible with interface
// sort.Interface.
func (o *OrderedSparse) Swap(i, j int) {
	o.Topics[i], o.Topics[j] = o.Topics[j], o.Topics[i]
	o.Counts[i], o.Counts[j] = o.Counts[j], o.Counts[i]
}

// Assign clears and recreates an OrderedSparse variable, and makes it
// represents s.
func (o *OrderedSparse) Assign(s Hist) *OrderedSparse {
	o.Topics = make([]int32, 0, s.Len())
	o.Counts = make([]int32, 0, s.Len())
	s.ForEach(func(topic int, count int64) error {
		o.Topics = append(o.Topics, int32(topic))
		o.Counts = append(o.Counts, int32(count))
		return nil
	})
	sort.Sort(o)
	return o
}

// AddDiff computes o += (m - s)
func (o *OrderedSparse) AddDiff(m, s *OrderedSparse) {
	tmp := NewSparse().AssignOrdered(o)
	for i, topic := range m.Topics {
		tmp[topic] += m.Counts[i]
	}
	for i, topic := range s.Topics {
		tmp[topic] -= s.Counts[i]
	}
	for topic, count := range tmp {
		if count == 0 {
			delete(tmp, topic)
		}
	}
	o.Assign(tmp)
}

// String prints an OrderedSparse variable the same format as a slice.
func (o OrderedSparse) String() string {
	out := "[ "
	for i, topic := range o.Topics {
		out += fmt.Sprintf("%d:%d ", topic, o.Counts[i])
	}
	out += "]"
	return out
}

// Count returns the count of a topic.
func (o OrderedSparse) At(topic int) int64 {
	for i := range o.Topics {
		if int(o.Topics[i]) == topic {
			return int64(o.Counts[i])
		}
	}
	return 0
}

// Inc increases the count of a topic.  It reallocates
// OrderedSparse.Topics and OrderedSparse.Counts if necessary.
func (o *OrderedSparse) Inc(topic, count int) {
	if topic < 0 {
		panic(fmt.Sprintf("topic (%d) < 0", topic))
	}
	if count <= 0 {
		panic(fmt.Sprintf("count (%d) <= 0", count))
	}
	if count > int(math.MaxInt32) {
		panic(fmt.Sprintf("count (%d) larger than MaxInt32", count))
	}

	// Increase an exisitng non-zero or append one.
	t := int32(topic)
	c := int32(count)
	var i int = 0
	for i < len(o.Topics) && o.Topics[i] != t {
		i++
	}
	if i < len(o.Topics) { // found
		if o.Counts[i] >= math.MaxInt32-t {
			panic(fmt.Sprintf("o[%d] = %d overflow", i, o.Counts[i]))
		}
		o.Counts[i] += c
	} else {
		o.Topics = append(o.Topics, t)
		o.Counts = append(o.Counts, c)
	}

	// Ensures that non-zeros are sorted in descending order.
	c = o.Counts[i]
	for i > 0 && c > o.Counts[i-1] {
		o.Topics[i], o.Counts[i] = o.Topics[i-1], o.Counts[i-1]
		i--
	}
	o.Topics[i] = t
	o.Counts[i] = c
}

// Dec decreases the count of a topic.  It might reslice
// OrderedSparse.Topics and OrderedSparse.Counts to reduce their
// len(), but it does not reallocate memory.
func (o *OrderedSparse) Dec(topic, count int) {
	if topic < 0 {
		panic(fmt.Sprintf("topic (%d) < 0", topic))
	}
	if count <= 0 {
		panic(fmt.Sprintf("count (%d) <= 0", count))
	}

	t := int32(topic)
	c := int32(count)
	var i int = 0
	for i < len(o.Topics) && o.Topics[i] != t {
		i++
	}
	if i >= len(o.Topics) {
		panic(fmt.Sprintf("topic %d does not exist", t))
	}
	if o.Counts[i] < c {
		panic(fmt.Sprintf("existing count (%d) < delta count (%d)",
			o.Counts[i], c))
	}
	o.Counts[i] -= c

	c = o.Counts[i]
	for i+1 < len(o.Topics) && c < o.Counts[i+1] {
		o.Topics[i], o.Counts[i] = o.Topics[i+1], o.Counts[i+1]
		i++
	}
	o.Topics[i] = t
	o.Counts[i] = c

	if c == 0 {
		o.Topics = o.Topics[:i]
		o.Counts = o.Counts[:i]
	}
}

// OrderedSparse.ForEach goes over elements in the order of descending count.
func (o *OrderedSparse) ForEach(p func(topic int, count int64) error) error {
	for i := 0; i < len(o.Topics); i++ {
		if e := p(int(o.Topics[i]), int64(o.Counts[i])); e != nil {
			return e
		}
	}
	return nil
}

// Clone creates a new OrderedSparse variable, makes it represents o.
func (o *OrderedSparse) Clone() Hist {
	n := NewOrderedSparse()
	n.Topics = make([]int32, len(o.Topics))
	n.Counts = make([]int32, len(o.Counts))
	copy(n.Topics, o.Topics)
	copy(n.Counts, o.Counts)
	return n
}
