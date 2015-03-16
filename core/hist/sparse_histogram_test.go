package hist

import (
	"reflect"
	"testing"
)

func TestNewSparse(t *testing.T) {
	s := NewSparse()
	if len(s) != 0 {
		t.Errorf("len(s): expected %d, got %d", 0, len(s))
	}
}

func TestSparseClear(t *testing.T) {
	s := Sparse{1: 2, 3: 5}
	s.Clear()
	if len(s) != 0 {
		t.Errorf("len(s): expected %d, got %d", 0, len(s))
	}
}

func TestSparseAssignOrdered(t *testing.T) {
	o := NewOrderedSparse().Assign(Sparse{1: 10})
	s := NewSparse().AssignOrdered(o)
	if len(s) != 1 {
		t.Error("len(s): expected: ", 1, " got: ", len(s))
	}
	if s[1] != 10 {
		t.Error("s[1], expected: ", 10, " got: ", s[1])
	}
}

func TestSparseAdd(t *testing.T) {
	s := Sparse{1: 10}
	s.Add(Sparse{1: 2})
	if s[1] != 12 {
		t.Error("s[1], expected: ", 12, " got: ", s[1])
	}
}

func TestSparseEqual(t *testing.T) {
	s1 := Sparse{1: 10}
	s2 := Sparse{1: 10}
	if !s1.Equal(s2) {
		t.Errorf("Equal(%v, %v): expected: true got: false\n", s1, s2)
	}

	s2.Clear()
	if s1.Equal(s2) {
		t.Errorf("Equal(%v, %v): expected: false got: true\n", s1, s2)
	}
}

func TestIncDec(t *testing.T) {
	s := Sparse{}
	s.Inc(2, 10)
	if len(s) != 1 {
		t.Errorf("Expecting len(s) = 1, got %d", len(s))
	}
	if s[2] != 10 {
		t.Errorf("Expecting s[2] = 10, got %d", s[2])
	}

	s.Dec(2, 5)
	if s[2] != 5 {
		t.Errorf("Expecting s[2] = 5, got %d", s[2])
	}

	s.Dec(2, 5)
	if len(s) != 0 {
		t.Errorf("Expecting len(s) = 0, got %d", len(s))
	}
}

func TestSparseClone(t *testing.T) {
	s := Sparse{}
	c := s.Clone()
	if c.Len() != 0 {
		t.Errorf("Expected %d, got %d", 0, c.Len())
	}

	s = Sparse{1: 2, 3: 4}
	c = s.Clone()
	if !reflect.DeepEqual(c, s) {
		t.Errorf("Expected %v, got %v", s, c)
	}
}
