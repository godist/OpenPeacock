package srv

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/wangkuiyi/file"
	"log"
	"strings"
)

// Config contains configuration information required by master.
type Config struct {
	// DeployDir and LogDir defines the directory where binaries and
	// log files are stored.
	DeployDir string
	LogDir    string

	// JobName will be used to identify the job in naming log files.
	JobName string

	// CorpusDir and VocabFile defines the input of a training jobs.
	// All shard files must be in an HDFS or local directory,
	// CorpusDir, which contains nothing other than shard files.
	// VocabFile does not define the mapping between token and id, but
	// Go type core.gibbs.Vocabulary does.
	CorpusDir string
	VocabFile string

	// Master is the address of master, e.g., "localhost:10000".
	Master string

	// Retry in starting processes.
	Retry int

	// Squads defines a set of squads, each must have the same
	// NumVShards.  Aggregator defines NumVShards aggregator
	// addresses, or is nil when there is only one squad and no
	// aggregation is required.  It is noticable that the number of
	// squads can be less than the number of data groups, e.g.,
	// NumShards/NumVShards.  In such cases, some squads might need to
	// have more than one groups of data.
	Squads      []Squad
	Aggregators []string

	// If both Squads and Aggregators are empty, master would derive
	// squads them heuristically from Machines and NumVShards.
	Machines   []string
	NumVShards int

	// JobDir is the directory containing all outputs of a training
	// job.
	JobDir string

	// Log-likelihood is computed after every LogllPeriod iterations.
	LogllPeriod int

	// Prior parameters
	NumTopics  int
	TopicPrior float64
	WordPrior  float64
}

// Squad defines the addresses of M loader instances and M sampler
// instances controlled by a coordinator.  A squad could optionally
// have a name.
type Squad struct {
	Name        string   // e.g., "squad1"
	Coordinator string   // e.g., "yiwang-ld:1900"
	Loaders     []string // e.g., "192.168.1.10:2021", "yiwang-ld:2020"
	Samplers    []string // e.g., "192.168.1.10:2021", "yiwang-ld:2020"
}

// The directory structure in JobDir is as:
//
//   JobDir
//       |-00000
//       |  \-model-0000x-of-0000y
//       |-00001
//       |  \-model-0000x-of-0000y
//       |  \-logll-0000x-of-0000y
//       |-00002
//       |  \-model-0000x-of-0000y
//       \-00003
//          |-model-0000x-of-0000y
//          \-logll-0000x-of-0000y
//
// Here we see that after every few iterations, we can have
// log-likelihood computed.  The logll file is a text file containing
// two numbers: the log-likelihood of a data shard and the number of
// words in that shard.  These two numbers make it easy to compute the
// perplexity.
//
// We do not need model files in gibbs/0000x except for few recent
// iterations, so we delete them to save disk space.  However, we do
// not delete the directory gibbs/0000x and gibbs/0000x/logll, so we
// can track the training progress.
//
// Notice that the estimated prior and the global topic histogram are
// duplicated and exist in every model shard file, as them exist in
// the memory space of every Sampler instance.
const (
	MODEL_FILE = "model"
	LOGLL_FILE = "logll"
)

func (c *Config) Validate() error {
	if len(c.JobName) <= 0 {
		return errors.New("c.JobName must be specified")
	}

	if len(c.Master) <= 0 {
		return errors.New("c.Master must be specified")
	}

	if len(c.Aggregators) <= 0 && len(c.Squads) <= 0 {
		msg := ""
		if len(c.Machines) <= 0 {
			msg += "c.Machines must not be emtpy. "
		}
		if c.NumVShards <= 0 {
			msg += "And c.NumVShards must be a positive value."
		}
		if len(msg) > 0 {
			return errors.New("With Squads and Aggregators empty, " + msg)
		} else {
			c.AllocateSquadsAndAggregators()
		}
	}

	if len(c.Aggregators) == 0 || len(c.Squads) == 0 {
		return errors.New("Either both c.Aggregators and c.Squads are " +
			" specified, or both empty.")
	}

	c.NumVShards = len(c.Aggregators)
	if c.NumVShards == 0 {
		return errors.New("c.NumVShards must be a postive value.")
	}

	msg := ""
	for i, s := range c.Squads {
		if len(s.Coordinator) <= 0 {
			msg += "s.Coordinator must be specified\n"
		}
		if len(s.Loaders) != c.NumVShards {
			msg += fmt.Sprintf("Squads[%d]: #Loaders != c.NumVShards\n", i)
		}
		if len(s.Samplers) != c.NumVShards {
			msg += fmt.Sprintf("Squads[%d]: #Samplers != c.NumVShards\n", i)
		}
	}

	if len(msg) > 0 {
		return errors.New(msg)
	}
	return nil
}

func (c *Config) AllocateSquadsAndAggregators() {
	log.Fatal("config.Config.AllocateSquadsAndAggregators not implemented yet")
}

// Encode returns the JSON-encoded Config, which can be used as the
// value of command line flag to pass information to sub-processes of
// master.
func (c *Config) Encode() (string, error) {
	var buf bytes.Buffer
	if e := json.NewEncoder(&buf).Encode(c); e != nil {
		return "", fmt.Errorf("JSON encoding failed: %v", e)
	}
	return buf.String(), nil
}

// String is required by interface flag.Var
func (c *Config) String() string {
	if b, e := json.MarshalIndent(c, " ", "  "); e == nil {
		return fmt.Sprintf("%s", b)
	}
	return ""
}

// Set is required by interface flag.Var.  It decode a JSON encoded
// Config variable.
func (c *Config) Set(value string) error {
	e := json.NewDecoder(strings.NewReader(value)).Decode(c)
	if e != nil {
		return fmt.Errorf("Error decoding JSON: %v", e)
	}
	return nil
}

// RegisterAsFlag registers a flag with name flagName and accepts a
// JSON encoded Config object as the value.  This function must be
// called before flag.Parse().
func (c *Config) RegisterAsFlag() {
	flag.Var(c, "config", "JSON encoded configuration")
}

func LoadConfig(filename string) (*Config, error) {
	f, e := file.Open(filename)
	if e != nil {
		return nil, fmt.Errorf("Cannot open config file %s: %v", filename, e)
	}
	defer f.Close()

	cfg := new(Config)
	if e = json.NewDecoder(f).Decode(cfg); e != nil {
		return nil, fmt.Errorf("Parse JSON config file: %v", e)
	}

	if e := cfg.Validate(); e != nil {
		return nil, fmt.Errorf("Invalid configuration: %v", e)
	}
	return cfg, nil
}

func (c *Config) SquadId(addr string) int {
	for i, s := range c.Squads {
		if s.Coordinator == addr {
			return i
		}
	}
	return -1
}

func (c *Config) SamplerId(coord, sampler string) (int, int) {
	if sid := c.SquadId(coord); sid >= 0 {
		for i, s := range c.Squads[sid].Samplers {
			if s == sampler {
				return sid, i
			}
		}
	}
	return -1, -1
}

func (c *Config) LoaderId(coord, loader string) (int, int) {
	if sid := c.SquadId(coord); sid >= 0 {
		for i, s := range c.Squads[sid].Loaders {
			if s == loader {
				return sid, i
			}
		}
	}
	return -1, -1
}

func (c *Config) AggregatorId(addr string) int {
	for i, a := range c.Aggregators {
		if a == addr {
			return i
		}
	}
	return -1
}
