package gibbs

import (
	"container/heap"
	"fmt"
	"math/rand"
	"testing"
)

func TestMinHeap(t *testing.T) {
	const K = 10
	const N = 100
	a := rand.Perm(N)
	h := newMinHeap(N)
	heap.Init(h)
	for _, i := range a {
		if len(*h) < K {
			heap.Push(h, wordFreq{i, int64(i)})
		} else if int64(i) > (*h)[0].freq {
			heap.Pop(h)
			heap.Push(h, wordFreq{i, int64(i)})
		}
	}

	for i := N - K; h.Len() > 0; i++ {
		if p := heap.Pop(h); p.(wordFreq).freq != int64(i) {
			t.Errorf("Expecting %v, got %v", wordFreq{i, int64(i)}, p)
		}
	}
}

func TestNewModelAccessor(t *testing.T) {
	m := CreateTestingModel()
	a := NewModelAccessor(m, 1)
	truth := "[[0.25 0.004901960784313725] [0.25 0.4950980392156863] [0.25 0.004901960784313725] [0.25 0.4950980392156863]]"
	if fmt.Sprint(a.WordTopicDists) != truth {
		t.Errorf("Expecting %s, got %s", truth, fmt.Sprint(a.WordTopicDists))
	}
}
