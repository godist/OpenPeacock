package main

import (
	"os"
	"os/exec"
	"path"
	"testing"
)

const (
	kTrain = `github.com/wangkuiyi/phoenix/cmd/interpreter/train_toy_model`
	kPrint = `github.com/wangkuiyi/phoenix/cmd/print_model`
)

func TestPrintModel(t *testing.T) {
	goCompiler, e := exec.LookPath("go")
	if e != nil {
		t.Fatalf("Cannot find go in PATH: %v", e)
	}

	goPath := os.Getenv("GOPATH")
	if len(goPath) <= 0 {
		t.Fatalf("GOPATH not set")
	}

	if e := exec.Command(goCompiler, "install", kTrain).Run(); e != nil {
		t.Fatalf("Cannot build %s: %v", kTrain, e)
	}

	if e := exec.Command(goCompiler, "install", kPrint).Run(); e != nil {
		t.Fatalf("Cannot build %s: %v", kPrint, e)
	}

	p, e := exec.Command(path.Join(goPath, "bin", path.Base(kTrain))).Output()
	if e != nil {
		t.Fatalf("Cannot run %s: %v", path.Base(kTrain), e)
	}
	dir := string(p)
	defer os.RemoveAll(dir)

	o, e := exec.Command(path.Join(goPath, "bin", path.Base(kPrint)),
		"-model="+path.Join(dir, "model"),
		"-vocab="+path.Join(dir, "vocab")).Output()
	if e != nil {
		t.Fatalf("Cannot run %s: %s, %v", path.Base(kPrint), o, e)
	}

	truth :=
		`Topic 00000 Nt 00004: orange (2) apple (2)
Topic 00001 Nt 00004: tiger (2) cat (2)
`
	if string(o) != truth {
		t.Errorf("Expected\n%s\ngot\n%s\n", truth, string(o))
	}
}
