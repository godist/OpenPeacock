// train_toy_model invokes core/gibbs.CreateTestingOptmizedModel to
// create a testing vocabulary and a testing model.  It writes these
// two data structures into temporary files, which can then be used to
// test cmd/interpreter.  It returns the name of a temporary
// directory, in which, the model and vocab files were generated.
package main

import (
	"encoding/gob"
	"fmt"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func main() {
	m, v, e := gibbs.CreateTestingOptimizedModel()
	if e != nil {
		log.Fatal("CreateTestingOptimizedModel failed:", e)
	}

	dir, e := ioutil.TempDir("", "train_toy_model")
	if e != nil {
		log.Fatal("Cannot create temp dir:", e)
	}

	if f, e := os.Create(path.Join(dir, "model")); e == nil {
		defer f.Close()
		enc := gob.NewEncoder(f)
		if e := enc.Encode(m); e != nil {
			log.Fatalf("Failed encoding model: %v", e)
		}
	} else {
		log.Fatalf("Cannot create model file: %v", e)
	}

	if f, e := os.Create(path.Join(dir, "vocab")); e == nil {
		defer f.Close()
		for _, token := range v.Tokens {
			fmt.Fprintf(f, "%s\n", token)
		}
	} else {
		log.Fatalf("Cannot create vocab file: %v", e)
	}

	fmt.Print(dir)
}
