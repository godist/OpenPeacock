package gibbs

import (
	"github.com/wangkuiyi/phoenix/core/hist"
)

// Optimizer collects statistics for optimizing the asymmetric
// Dirichlet topic prior and the symmetric Dirichlet word prior.
type Optimizer struct {
	// docLenHist is for esitmating topic prior.  It is the histogram
	// of document lengths.
	docLenHist hist.Sparse
	// topicDocHists[t] is a histogram of the number of documents, in
	// which topic k occurs n times.
	topicDocHists []hist.Sparse
}

func NewOptimizer(numTopic int) *Optimizer {
	o := &Optimizer{
		docLenHist:    hist.NewSparse(),
		topicDocHists: make([]hist.Sparse, numTopic),
	}
	for i := range o.topicDocHists {
		o.topicDocHists[i] = hist.NewSparse()
	}
	return o
}

func (o *Optimizer) CollectDocumentStatistics(d *Document) {
	for i := 0; i < d.TopicHist.Len(); i++ {
		h := o.topicDocHists[d.TopicHist.Topics[i]]
		h[int32(d.TopicHist.Counts[i])]++
	}
	o.docLenHist[int32(d.Len())]++
}

// approximateHist creates a dense histogram that approximates a
// sparse histogram.  The length of the histogram is the maximum index
// value in the sparse histogram.  This function is only used to
// compute the Digamma function used in prior optimization.
func approximateHist(s hist.Sparse) hist.Dense {
	if len(s) == 0 {
		return nil
	}

	var maxIdx int32 = 0
	for k := range s {
		if k > maxIdx {
			maxIdx = k
		}
	}

	d := hist.NewDense(int(maxIdx) + 1)
	s.ForEach(func(k int, v int64) error {
		d.Inc(k, int(v))
		return nil
	})
	return d
}

// OptimizeTopicPriors optimizes asymmetic Dirichlet-Multinomial
// hyperparameters using Minka's fixed-point iteration and the
// digamma recurrence relation, as described in
//   Hanna M. Wallach. Structured Topic Models for Language. Ph.D.
//   thesis, University of Cambridge, 2008.
func (o *Optimizer) OptimizeTopicPriors(
	m *Model, shape, scale float64, iterations int) {
	for it := 0; it < iterations; it++ {
		diff_digamma, denominator := 0.0, 0.0
		d := approximateHist(o.docLenHist)
		for i := 1; i < len(d); i++ {
			diff_digamma += 1.0 / (float64(i) - 1.0 + m.TopicPriorSum)
			denominator += float64(d[i]) * diff_digamma
		}
		denominator -= 1.0 / scale

		m.TopicPriorSum = 0.0
		for k, h := range o.topicDocHists {
			diff_digamma, numerator := 0.0, 0.0
			d := approximateHist(h)
			for i := 1; i < len(d); i++ {
				diff_digamma += 1.0 / (float64(i) - 1.0 + m.TopicPrior[k])
				numerator += float64(d[i]) * diff_digamma
			}
			m.TopicPrior[k] = (m.TopicPrior[k]*numerator + shape) / denominator
			m.TopicPriorSum += m.TopicPrior[k]
		}
	}
}
