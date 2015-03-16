package utils

import (
	"bytes"
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/plotutil"
	"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/vg/vgimg"
	"expvar"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"
)

type Iteration struct {
	StartTime  time.Time
	Duration   time.Duration
	Perplexity float64
}
type Iterations []*Iteration

func (is *Iterations) String() string { // Implements expvar.Var
	var buf bytes.Buffer
	for i, iter := range *is {
		fmt.Fprintf(&buf, "%05d: %s\t%s\n", i, iter.StartTime, iter.Duration)
	}
	return buf.String()
}

func (is *Iterations) Start() *Iteration {
	i := &Iteration{StartTime: time.Now()}
	*is = append(*is, i)
	return i
}

func (is *Iterations) End(perplexity float64) *Iteration {
	i := (*is)[len(*is)-1]
	i.Duration = time.Since(i.StartTime)
	i.Perplexity = perplexity
	return i
}

func EnableExpvar(addr string) *Iterations {
	is := new(Iterations)
	*is = make(Iterations, 0)

	expvar.Publish("Iterations", is)
	http.Handle("/progress/perplexity", newPerplexityFigureHandler(is))
	http.Handle("/progress/duration", newDurationFigureHandler(is))

	go func() {
		if e := http.ListenAndServe(addr, nil); e != nil {
			log.Fatalf("ListenAndServe on %s failed: %v", addr, e)
		}
	}()

	return is
}

func newPerplexityFigureHandler(is *Iterations) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ps := make(plotter.XYs, 0, len(*is))
		for i, _ := range *is {
			if (*is)[i].Perplexity > 0.0 {
				ps = append(ps,
					struct{ X, Y float64 }{float64(i), (*is)[i].Perplexity})
			}
		}
		if e := plotFigure(w, ps, "Iteration", "Perplexity"); e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
		}
	}
}

func newDurationFigureHandler(is *Iterations) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ps := make(plotter.XYs, 0, len(*is))
		for i, _ := range *is {
			if i > 0 && (*is)[i].Duration > 0 {
				// Skip the initialization and yet-complete iterations.
				ps = append(ps, struct{ X, Y float64 }{
					float64(i), (*is)[i].Duration.Minutes()})
			}
		}
		if e := plotFigure(w, ps, "Iteration", "Duration"); e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
		}
	}
}

// The following code snippet is largely copied from
// https://code.google.com/p/plotinum/issues/detail?id=122
func plotFigure(w io.Writer, ps plotter.XYs, xLabel, yLabel string) error {
	p, e := plot.New()
	if e != nil {
		return fmt.Errorf("plot.New failed: %v", e)
	}

	p.Title.Text = strings.Join(os.Args, " ")
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel

	p.Add(plotter.NewGrid())
	if e := plotutil.AddLinePoints(p, "", ps); e != nil {
		return fmt.Errorf("plotutil.AddLinePoints failed: %v", e)
	}

	c := vgimg.PngCanvas{vgimg.New(vg.Length(640), vg.Length(480))}
	p.Draw(plot.MakeDrawArea(c))
	c.WriteTo(w)
	return nil
}
