package hist

import (
	"fmt"
	"testing"
)

func TestNewOrderedSparseInc(t *testing.T) {
	m := NewOrderedSparse()
	if m.Len() != 0 {
		t.Errorf("Expecting m.Len() = 0, got %d", m.Len())
	}

	m = NewOrderedSparseAndReserve(1)
	if m.Len() != 0 {
		t.Errorf("Expecting m.Len() = 0, got %d", m.Len())
	}
}

func TestOrderedSparseAssign(t *testing.T) {
	o := NewOrderedSparse().Assign(Sparse{})
	str := "[ ]"
	if fmt.Sprint(o) != str {
		t.Errorf("Expected %s, got %v", str, o)
	}

	o = NewOrderedSparse().Assign(Sparse{0: 7, 1: 2, 2: 1, 3: 10})
	str = "[ 3:10 0:7 1:2 2:1 ]"
	if fmt.Sprint(o) != str {
		t.Errorf("Expected %s, got %v", str, o)
	}
}

func TestOrderedSparseAddDiff(t *testing.T) {
	m := NewOrderedSparse().Assign(Sparse{1: 1, 2: 1})
	s := NewOrderedSparse().Assign(Sparse{0: 1, 2: 1})
	o := NewOrderedSparse().Assign(Sparse{0: 1})
	o.AddDiff(m, s)
	str := "[ 1:1 ]"
	if fmt.Sprint(o) != str {
		t.Errorf("Expected %s, got %v", str, o)
	}
}

func TestOrderedSparseCount(t *testing.T) {
	m := NewOrderedSparse().Assign(Sparse{1: 2, 2: 1})
	if m.At(1) != 2 {
		t.Errorf("Expecting m.At(1) = 2, got %d", m.At(1))
	}
	if m.At(2) != 1 {
		t.Errorf("Expecting m.At(2) = 1, got %d", m.At(2))
	}
	if m.At(0) != 0 {
		t.Errorf("Expecting m.At(0) = 0, got %d", m.At(2))
	}
}

func TestOrderedSparseInc(t *testing.T) {
	// Test with various amount of reservation.
	for reserved := 0; reserved < 3; reserved++ {
		m := NewOrderedSparseAndReserve(reserved)
		nonzero := 2*reserved + 1
		for t := 0; t < nonzero; t++ {
			m.Inc(t, t+1)
			m.Inc(t, t+1) // increase an existing non-zero
		}
		for i := 0; i < nonzero; i++ {
			if m.Topics[i] != int32(nonzero-1-i) {
				t.Errorf("Expecting m.Topics[%d] = %d, got %d",
					i, nonzero-1-i, m.Topics[i])
			}
			if m.Counts[i] != 2*int32(nonzero-i) {
				t.Errorf("Expecting m.Counts[%d] = %d, got %d",
					i, nonzero-i, m.Counts[i])
			}
		}
	}
}

func TestOrderedSparseDec(t *testing.T) {
	m := NewOrderedSparse().Assign(Sparse{0: 1, 1: 2})
	m.Dec(1, 1)
	if fmt.Sprint(m) != "[ 1:1 0:1 ]" {
		t.Errorf("Expecting m = [ 1:1 0:1 ], got %v", m)
	}
	m.Dec(0, 1)
	if fmt.Sprint(m) != "[ 1:1 ]" {
		t.Errorf("Expecting m = [ 1:1 ], got %v", m)
	}
	m.Dec(1, 1)
	if fmt.Sprint(m) != "[ ]" {
		t.Errorf("Expecting m = [ ], got %v", m)
	}

	// In another order of non-zeros.
	m = NewOrderedSparse().Assign(Sparse{0: 1, 1: 2})
	m.Dec(1, 2)
	m.Dec(0, 1)
	if fmt.Sprint(m) != "[ ]" {
		t.Errorf("Expecting m = [ ], got %v", m)
	}
}

func TestOrderedSparseClone(t *testing.T) {
	str := "[ 2:8 3:5 1:2 0:1 ]"
	o := NewOrderedSparse().Assign(Sparse{0: 1, 1: 2, 3: 5, 2: 8})
	c := o.Clone()
	if fmt.Sprint(c) != str {
		t.Errorf("Expected %s, got %v", str, c)
	}

	o = NewOrderedSparse()
	c = o.Clone()
	if c.Len() != 0 {
		t.Errorf("Expected %d, got %d", 0, c.Len())
	}
}
