package gibbs

import (
	"fmt"
	"testing"
)

const (
	testingDocument = "&{[ 1:2 ] [3 1] [1 1]}"
)

func TestInitializeDocument(t *testing.T) {
	v, e := CreateTestingVocabulary()
	if e != nil {
		t.Errorf("Failed building testing vocabulary")
	}

	d := CreateTestingDocument(v)
	if fmt.Sprint(d) != testingDocument {
		t.Errorf("Expecting d = %s, Got %s", testingDocument, fmt.Sprint(d))
	}
}
