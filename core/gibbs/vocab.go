package gibbs

import (
	"bufio"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"sort"
	"strings"
)

// Vocabulary maintains the bi-directional mapping between strings and
// ids.  Ids are assigned to strings randomly and are in the range of
// [0, N), where N is the vocabulary size.  This mapping is stored as
// a sorted slice of strings.  The order of a token becomes it ID.
// Sorting order is the ascending order of string hashes + lexical
// order; thus shuffles highly-frequent and long-tail tokens and
// balancing workloads when we distribute the vocabulary.
type Vocabulary struct {
	Tokens []string
	hasher hash.Hash64
	ids    map[string]int
}

func NewVocabulary() *Vocabulary {
	return &Vocabulary{
		Tokens: make([]string, 0),
		hasher: fnv.New64a(),
	}
}

func (v *Vocabulary) Load(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fs := strings.Fields(scanner.Text())
		if len(fs) > 0 {
			v.Tokens = append(v.Tokens, fs[0]) // Take only the first column.
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	sort.Sort(v)
	v.buildIdMap()
	return nil
}

func (v *Vocabulary) buildIdMap() {
	v.ids = make(map[string]int)
	for i, _ := range v.Tokens {
		v.ids[v.Tokens[i]] = i
	}
}

func (v *Vocabulary) Len() int {
	return len(v.Tokens)
}

// fingerPrint returns the FNV-1a hash of the i-th token.
func (v *Vocabulary) fingerprint(s string) uint64 {
	v.hasher.Write([]byte(s))
	sum := v.hasher.Sum64()
	v.hasher.Reset()
	return sum
}

func (v *Vocabulary) Less(i, j int) bool {
	l, r := v.fingerprint(v.Tokens[i]), v.fingerprint(v.Tokens[j])
	if l == r {
		return v.Tokens[i] < v.Tokens[j]
	}
	return l < r
}

func (v *Vocabulary) Swap(i, j int) {
	v.Tokens[i], v.Tokens[j] = v.Tokens[j], v.Tokens[i]
}

func (v *Vocabulary) Token(id int32) string {
	if int(id) < 0 || int(id) >= len(v.Tokens) {
		panic(fmt.Sprintf("id=%d out of range [0, %d)", id, len(v.Tokens)))
	}
	return v.Tokens[id]
}

// Id returns the index of token.  If token is not in the vocabulary,
// it returns a negative value.
func (v *Vocabulary) Id(token string) int32 {
	if v.ids == nil {
		v.buildIdMap()
	}
	if id, ok := v.ids[token]; ok {
		return int32(id)
	}
	return int32(-1)
}
