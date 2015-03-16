package gibbs

import (
	"math"
)

// Evaluator computes log-likelihood and perplexity of documents given
// a model.
//
// It implements an accelerated algorithm proposed by Xuemin Zhao
// (xmzhao1986@qq.com).  The algorithm caches pre-computable factors
// when initialize Evaluator.  The idea of extracting pre-computable
// factors can be perceived from the following equation on the
// likelihood of a token (t) in document m:
/*
                                n_mk + a_k
   L(m,t)  = \sum_k \phi_kt  ------------------
                              L_m + \sum_k a_k

                     1          [                                          ]
           = -----------------  [ \sum_k \phi_kt a_k + \sum_k \phi_kt n_mk ]
             L_m + \sum_k a_k   [                                          ]

                     1          [                            ]
           = -----------------  [ o(t) + \sum_k \phi_kt n_mk ]
             L_m + \sum_k a_k   [                            ]
*/
// Above method runs fast for the following reasons:
/*
   1. o(t) = \sum_k \phi_kt a_k can be pre-computed and cached.
   2. \sum_k \phi_kt n_mk can be computed quickly as n_mk is a sparse vector.
   3. \sum_k a_k has been cached in Model.TopicPriorSum
   4. L_m is just document length.
*/
// More than that, the computing of o(t) can be further accelerated by
// using Sampler.smoothingOnlyBucketSize, aka s, which is a scalar
// value:
/*
                 a_k b
   s = \sum_k ------------
                bV + N_t
*/
// Considering that
/*
   o(t) = \sum_k \phi_kt a_k

                  (b + n_kt)
        = \sum_k ------------- a_k
                   bV + N_t

                   a_k b               a_k n_kt
        = \sum_k ---------- + \sum _k ----------
                  bV + N-t             bV + N-t

                        a_k n_kt
        = s +  \sum _k ----------
                        bV + N-t
*/
// Because n_kt is a sparse vector, the last \sum_k in above equation
// take O(#non-zeros) instead of O(K).
type Evaluator struct {
	model       *ModelAccessor
	cachedCoeff []float64
}

func NewEvaluator(model *Model, cacheSizeMB int, s *Sampler) *Evaluator {
	accessor := NewModelAccessor(model, cacheSizeMB)
	return &Evaluator{
		model:       accessor,
		cachedCoeff: calculateEvaluationCoeff(accessor, s),
	}
}

func calculateEvaluationCoeff(model *ModelAccessor, s *Sampler) []float64 {
	coeff := make([]float64, len(model.WordTopicHists))
	// TODO(yi): Parallellize the following loop.
	for token, _ := range model.WordTopicHists {
		if hist := model.WordTopicHists[token]; hist != nil {
			if s != nil {
				hist.ForEach(func(topic int, count int64) error {
					coeff[token] +=
						model.TopicPrior[topic] * float64(count) /
							(model.WordPrior +
								float64(model.GlobalTopicHist.At(topic)))
					return nil
				})
				coeff[token] += s.smoothingOnlyBucketSize
			} else {
				dist := model.WordTopicDist(int32(token))
				for j, _ := range dist {
					coeff[token] += dist[j] * model.TopicPrior[j]
				}
			}
		}
	}
	return coeff
}

// Perplexity computes log-likelihood of a document. It returns
// log-likelihood as well as the document length, which, when divided,
// get to the perplexity of the document, or when aggregated along
// documents then divided, get to the perplexity of corpus.
func (e *Evaluator) Perplexity(doc *Document) (float64, int) {
	if doc.Len() <= 0 {
		return 0.0, 0
	}

	logl := 0.0
	cache := newDistCache(e.model)
	for i := 0; i < doc.Len(); i++ {
		wordTopicDist := cache.Get(doc.Words[i])
		prob := 0.0
		doc.TopicHist.ForEach(func(topic int, count int64) error {
			prob += wordTopicDist[topic] * float64(count)
			return nil
		})
		logl += math.Log((e.cachedCoeff[doc.Words[i]] + prob) /
			(float64(doc.Len()) + e.model.TopicPriorSum))
	}
	return logl, doc.Len()
}
