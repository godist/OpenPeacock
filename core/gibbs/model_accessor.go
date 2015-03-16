package gibbs

import (
	"container/heap"
	"unsafe"
)

type ModelAccessor struct {
	*Model
	WordTopicDists [][]float64
	smoothingOnly  []float64
}

// Creates a ModelAccessor instance, which contains
// ModelAccessor.WordTopicDists converted from Model.WordTopicHists,
// but might not include all topic distributions as memory space is
// constrained by cacheSizeMB.  However, if cacheSizeMB is a negative,
// topic distributions of all words in the vocabulary will be
// included.
func NewModelAccessor(model *Model, cacheSizeMB int) *ModelAccessor {
	a := &ModelAccessor{
		model,
		make([][]float64, model.VocabSize()),
		nil}

	// The maximum number C of topic distributions that can be cached.
	cached := model.VocabSize()
	if cacheSizeMB >= 0 {
		var f64 float64
		cached = (cacheSizeMB*1024*1024 -
			model.VocabSize()*int(unsafe.Sizeof(a.WordTopicDists[0]))) /
			(model.NumTopics() * int(unsafe.Sizeof(f64)))
	}

	if cached > 0 {
		// Count the word frequncies and select the largest C words.
		h := newMinHeap(len(model.WordTopicHists))
		heap.Init(h)
		for word, hist := range model.WordTopicHists {
			var freq int64
			if hist != nil {
				hist.ForEach(func(_ int, count int64) error {
					freq += count
					return nil
				})
			}

			if len(*h) < cached {
				heap.Push(h, wordFreq{word, freq})
			} else if freq > (*h)[0].freq {
				heap.Pop(h)
				heap.Push(h, wordFreq{word, freq})
			}
		}

		// Cache topic distributions of the largest C words.
		for h.Len() > 0 {
			wf := heap.Pop(h).(wordFreq)
			dist := a.buildSmoothingOnly()
			a.cumulatePosterior(dist, int32(wf.word))
			a.WordTopicDists[wf.word] = dist
		}
	}
	return a
}

func (a *ModelAccessor) buildSmoothingOnly() []float64 {
	if len(a.smoothingOnly) <= 0 {
		dist := make([]float64, a.NumTopics())
		a.GlobalTopicHist.ForEach(func(topic int, count int64) error {
			dist[topic] = a.WordPrior / (a.WordPriorSum + float64(count))
			return nil
		})
		a.smoothingOnly = dist
	}

	dist := make([]float64, a.NumTopics())
	copy(dist, a.smoothingOnly)
	return dist
}

func (a *ModelAccessor) cumulatePosterior(dist []float64, token int32) {
	hist := a.WordTopicHists[token]
	if hist != nil {
		hist.ForEach(func(t int, c int64) error {
			dist[t] = (float64(c) + a.WordPrior) /
				(a.WordPriorSum + float64(a.GlobalTopicHist.At(t)))
			return nil
		})
	}
}

func (a *ModelAccessor) WordTopicDist(token int32) []float64 {
	if dist := a.WordTopicDists[token]; dist != nil {
		return dist
	}

	dist := a.buildSmoothingOnly()
	a.cumulatePosterior(dist, token)
	return dist
}

type minHeap []wordFreq
type wordFreq struct {
	word int
	freq int64
}

func newMinHeap(size int) *minHeap {
	h := new(minHeap)
	*h = make(minHeap, 0, size)
	return h
}

func (h minHeap) Len() int            { return len(h) }
func (h minHeap) Less(i, j int) bool  { return h[i].freq < h[j].freq }
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) { *h = append(*h, x.(wordFreq)) }
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
