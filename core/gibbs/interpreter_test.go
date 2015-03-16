package gibbs

import (
	"fmt"
	"testing"
)

func TestInterpret(t *testing.T) {
	m, v, e := CreateTestingOptimizedModel()
	if e != nil {
		t.Skip(e)
	}

	intr := NewInterpreter(m, v, -1)

	type testCase struct {
		doc  []string
		err  string
		dist string
	}
	testCases := []testCase{
		{[]string{}, ErrEmptyDoc, ""},
		{[]string{"tiger"}, "", "[{1 1}]"},
		{[]string{"cat"}, "", "[{1 1}]"},
		{[]string{"apple"}, "",
			"[{0 0.9795918367346939} {1 0.02040816326530612}]"},
		{[]string{"orange"}, "", "[{0 1}]"},
		{[]string{"unknown"}, ErrEmptyDoc, ""},
		{[]string{"unknown", "apple"}, "", "[{0 1}]"},
		{[]string{"tiger", "cat"}, "", "[{1 1}]"},
		{[]string{"cat", "tiger"}, "", "[{1 1}]"},
		{[]string{"apple", "orange"}, "", "[{0 1}]"},
		{[]string{"orange", "apple"}, "", "[{0 1}]"},
		{[]string{"tiger", "apple"}, "",
			"[{0 0.5306122448979592} {1 0.46938775510204084}]"},
	}
	for _, c := range testCases {
		d, e := intr.Interpret(c.doc, 50, 100)
		if e != nil {
			if fmt.Sprint(e) != c.err {
				t.Errorf("%s: Expect error \"%s\", got \"%s", c.doc, c.err, e)
			}
		} else if c.dist != fmt.Sprint(d) {
			t.Error(c.doc, ":", "Expecting result", c.dist, "got", d)
		}
	}
}
