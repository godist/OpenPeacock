package srv

import (
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/file/inmemfs"
	"path"
	"testing"
)

func TestMasterTaskInitialization(t *testing.T) {
	c := createTestingConfig()
	c.Validate() // This sets c.NumVShards

	inmemfs.Format()
	for i := 0; i < c.NumVShards+1; i++ {
		f, e := file.Create(path.Join(c.CorpusDir, fmt.Sprintf("%05d", i)))
		if e != nil {
			t.Errorf("Unexpected error in create file: %v", e)
		}
		f.Write([]byte(fmt.Sprintf("Hello %d", i)))
		f.Close()
	}

	m, e := NewMaster(c, nil)
	if e != nil {
		t.Fatalf("Unexpected error: %v", e)
	}

	if len(m.pending) != 2 {
		t.Errorf("Expecting len(m.pending) = %d, got %d", 2, len(m.pending))
	}
	if len(m.pending[0].Shards) != c.NumVShards {
		t.Errorf("Expecting the first task contains %d shards, got %d",
			c.NumVShards, len(m.pending[0].Shards))
	}
	if len(m.pending[1].Shards) != 1 {
		t.Errorf("Expecting the second task contains 1 shards, got %d",
			len(m.pending[1].Shards))
	}
}
