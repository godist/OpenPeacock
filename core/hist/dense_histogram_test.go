package hist

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNewDense(t *testing.T) {
	h := NewDense(2)
	h_str := "[0 0]"
	if h_str != fmt.Sprint(h) {
		t.Error("NewDense(2), expected", h_str, "got", h)
	}
}

func TestDenseClone(t *testing.T) {
	s := NewDense(0)
	c := s.Clone()
	if c.Len() != 0 {
		t.Errorf("Expected %v, got %v", s, c)
	}

	s = Dense{2, 0}
	c = s.Clone()
	if !reflect.DeepEqual(s, c) {
		t.Errorf("Expected %v, got %v", s, c)
	}
}
