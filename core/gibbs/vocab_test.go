package gibbs

import (
	"testing"
)

const (
	testingAppleFP  uint64 = 17819163333647859135
	testingOrangeFP uint64 = 12023831162993772011
)

func TestVocabularyFingerprint(t *testing.T) {
	v := NewVocabulary()
	if v.fingerprint("apple") != testingAppleFP {
		t.Errorf("Expecting fingerprint(\"apple\") = %d, got %d",
			testingAppleFP, v.fingerprint("apple"))
	}
	if v.fingerprint("apple") != testingAppleFP {
		t.Errorf("Expecting fingerprint(\"apple\") = %d, got %d",
			testingAppleFP, v.fingerprint("apple"))
	}

	if v.fingerprint("orange") != testingOrangeFP {
		t.Errorf("Expecting fingerprint(\"orange\") = %d, got %d",
			testingOrangeFP, v.fingerprint("orange"))
	}
}

func TestVocabularyLoad(t *testing.T) {
	v, e := CreateTestingVocabulary()
	if e != nil {
		t.Errorf("Load failed: %v", e)
	}

	if v.Len() != testingV {
		t.Errorf("Expecting v.Len() = %d, got %d", testingV, v.Len())
	}
}

func TestVocabularyTokenAndId(t *testing.T) {
	v, e := CreateTestingVocabulary()
	if e != nil {
		t.Errorf("Load failed: %v", e)
	}

	if v.Id("apple") != 3 {
		t.Errorf("Expecting v.Id(\"apple\") = 3, got %d", v.Id("apple"))
	}
	if v.Id("orange") != 1 {
		t.Errorf("Expecting v.Id(\"orange\") = 1, got %d", v.Id("orange"))
	}
	if v.Id("cat") != 2 {
		t.Errorf("Expecting v.Id(\"cat\") = 2, got %d", v.Id("cat"))
	}
	if v.Id("tiger") != 0 {
		t.Errorf("Expecting v.Id(\"tiger\") = 0, got %d", v.Id("tiger"))
	}

	if v.Id("unknown") != -1 {
		t.Errorf("Expecting v.Id(\"unknown\") = -1, got %d", v.Id("unknown"))
	}

	if v.Token(3) != "apple" {
		t.Errorf("Expecting v.Token(3) = \"apple\", got %s", v.Token(2))
	}
	if v.Token(1) != "orange" {
		t.Errorf("Expecting v.Token(1) = \"orange\", got %s", v.Token(0))
	}
	if v.Token(2) != "cat" {
		t.Errorf("Expecting v.Token(2) = \"cat\", got %s", v.Token(1))
	}
	if v.Token(0) != "tiger" {
		t.Errorf("Expecting v.Token(0) = \"tiger\", got %s", v.Token(1))
	}
}
