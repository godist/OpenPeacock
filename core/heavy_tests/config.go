package heavy_tests

const (
	// In the following soruce path, there is a real big model and
	// related vocab suitable for performance benchmark.
	kBigData = "src/github.com/wangkuiyi/phoenix/core/heavy_tests/testdata"
	kVocab   = "vocab.bz2"
	kModel   = "model.bz2"

	kAlpha     = 0.1
	kBeta      = 0.01
	kOptimIter = 5
	kShape     = 0.0
	kScale     = 1e7
)
