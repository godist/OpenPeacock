package hist

type Hist interface {
	At(topic int) int64
	Inc(topic, count int)
	Dec(topic, count int)
	Len() int

	// ForEach access elements in the histogram one-by-one. For each
	// element <topic, count>, it calls p(topic, count).  If p returns
	// nil, it goes on to rest elements; otherwise, it stops the
	// traversal and returns the error from p.
	ForEach(p func(topic int, count int64) error) error

	Clone() Hist
}
