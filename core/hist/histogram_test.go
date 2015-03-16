package hist

import (
	"errors"
	"fmt"
	"testing"
)

func ExampleHist(h Hist, exp string, t *testing.T) error {
	h.Inc(0, 1)
	h.Inc(1, 2)

	l := 0
	if e := h.ForEach(func(topic int, count int64) error {
		if topic+1 != int(count) {
			return errors.New("Wrong content")
		}
		l++
		return nil
	}); e != nil {
		return fmt.Errorf("Unexpected error: %v", e)
	}
	if l != h.Len() {
		return fmt.Errorf("Expecting len=%d, got %d", h.Len(), l)
	}

	if e := h.ForEach(func(topic int, count int64) error {
		return errors.New(fmt.Sprintf("%d %d ", topic, count))
	}); fmt.Sprint(e) != exp {
		return fmt.Errorf("Expecting %s; got: %v", exp, e)
	}

	return nil
}

func TestDenseIsHist(t *testing.T) {
	var d Hist
	d = NewDense(2)
	if e := ExampleHist(d, "0 1 ", t); e != nil {
		t.Errorf("%v", e)
	}
}

func TestSparseIsHist(t *testing.T) {
	var s Hist
	s = NewSparse()
	if e := ExampleHist(s, "0 1 ", t); e != nil {
		t.Errorf("%v", e)
	}
}

func TestOrderedSparseIsHist(t *testing.T) {
	var o Hist
	o = NewOrderedSparse()
	if e := ExampleHist(o, "1 2 ", t); e != nil {
		t.Errorf("%v", e)
	}
}
