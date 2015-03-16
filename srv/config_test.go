package srv

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
)

func createTestingConfig() *Config {
	return &Config{
		JobName:   "unittest",
		CorpusDir: "inmem:/usr/phoenix/corpus/",
		VocabFile: "inmem:/usr/phoenix/vocab",
		JobDir:    "inmem:/usr/unittest",
		Master:    "vm0:10000",
		Squads: []Squad{
			Squad{
				Name:        "squad0",
				Coordinator: "vm0:10010",
				Loaders:     []string{"vm0:10020", "vm1:10021"},
				Samplers:    []string{"vm2:10030", "vm3:10031"}},
			Squad{
				Name:        "squad1",
				Coordinator: "vm1:10011",
				Loaders:     []string{"vm0:10023", "vm1:10024"},
				Samplers:    []string{"vm2:10032", "vm3:10033"}},
		},
		Aggregators: []string{"vm0:10040", "vm1:10041"},
	}
}

func TestConfigJsonCodec(t *testing.T) {
	c := createTestingConfig()
	var buf bytes.Buffer
	e := json.NewEncoder(&buf).Encode(c)
	if e != nil {
		t.Errorf("Failed in encoding: %v", e)
	}

	d := json.NewDecoder(strings.NewReader(buf.String()))
	var c1 Config
	if e := d.Decode(&c1); e != nil {
		t.Errorf("Failed in decoding: %v", e)
	}

	b, _ := json.Marshal(c)
	b1, _ := json.Marshal(c1)
	if !bytes.Equal(b, b1) {
		t.Errorf("Encoded and decoded JSON does not equal to the original")
	}
}

func TestConfigValidate(t *testing.T) {
	c := createTestingConfig()
	if e := c.Validate(); e != nil {
		t.Errorf("Unexpected error from Config.Validate(): %v", e)
	}

	c.Squads = nil
	if e := c.Validate(); e == nil {
		t.Errorf("Expecting an error but got none")
	}

	c = createTestingConfig()
	c.Aggregators = nil
	if e := c.Validate(); e == nil {
		t.Errorf("Expecting an error but got none")
	}
}

func TestConfigArgs(t *testing.T) {
	c := createTestingConfig()
	f, e := c.Encode()
	if e != nil {
		t.Errorf("Failed encode config.Config")
	}
	os.Args = make([]string, 2)
	os.Args[1] = "-config=" + f
	var c1 Config
	c1.RegisterAsFlag()
	flag.Parse()

	en1, _ := c1.Encode()
	en2, _ := c.Encode()
	if en1 != en2 {
		t.Errorf("Decoded an encoded Coordinator %s not consistent with %s",
			en1, en2)
	}
}

func TestConfigSquadId(t *testing.T) {
	c := createTestingConfig()
	if c.SquadId("vm0:10010") != 0 {
		t.Errorf("Expecting 0, got %d", c.SquadId("vm0:10010"))
	}
	if c.SquadId("vm1:10011") != 1 {
		t.Errorf("Expecting 1, got %d", c.SquadId("vm1:10011"))
	}
}

func TestConfigSamplerId(t *testing.T) {
	c := createTestingConfig()
	if c, s := c.SamplerId("vm0:10010", "vm2:10030"); c != 0 || s != 0 {
		t.Errorf("Expecting c=%d, s=%d; got c=%d, s=%d", 0, 0, c, s)
	}
	if c, s := c.SamplerId("vm1:10011", "vm3:10033"); c != 1 || s != 1 {
		t.Errorf("Expecting c=%d, s=%d; got c=%d, s=%d", 1, 1, c, s)
	}
}

func TestConfigLoaderId(t *testing.T) {
	c := createTestingConfig()
	if c, s := c.LoaderId("vm0:10010", "vm0:10020"); c != 0 || s != 0 {
		t.Errorf("Expecting c=%d, s=%d; got c=%d, s=%d", 0, 0, c, s)
	}
	if c, s := c.LoaderId("vm1:10011", "vm1:10024"); c != 1 || s != 1 {
		t.Errorf("Expecting c=%d, s=%d; got c=%d, s=%d", 1, 1, c, s)
	}
}
