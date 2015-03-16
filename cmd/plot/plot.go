package main

import (
	"bufio"
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/plotutil"
	"flag"
	"github.com/wangkuiyi/compress_io"
	"log"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

func main() {
	flagLog := flag.String("log", "", "The log file")
	flagDocLen := flag.String("doclen", "", "The doc-length file")
	flagVocab := flag.String("vocab", "", "The vocab file")
	flagOut := flag.String("outdir", "", "Output directory")
	flag.Parse()

	var wg sync.WaitGroup
	outFile := func(dir, inFile string) string {
		return path.Join(dir, path.Base(inFile)+".pdf")
	}
	if len(*flagLog) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plotLog(*flagLog, outFile(*flagOut, *flagLog))
		}()
	}
	if len(*flagDocLen) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plotDocLen(*flagDocLen, outFile(*flagOut, *flagDocLen))
		}()
	}
	if len(*flagVocab) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plotVocab(*flagVocab, outFile(*flagOut, *flagVocab))
		}()
	}
	wg.Wait()
}

func plotLog(logFile, imageFile string) {
	f, e := os.Open(logFile)
	if e != nil {
		log.Fatalf("Cannot open input file %s: %v", logFile, e)
	}
	defer f.Close()

	log.Printf("Loading log file: %s ...", logFile)
	re := regexp.MustCompile(".*Iteration ([0-9]+) perplexity ([0-9\\.]+)$")
	pts := make(plotter.XYs, 0)
	maxInt := int(^uint(0) >> 1)
	prevIter := maxInt

	s := bufio.NewScanner(f)
	for s.Scan() {
		ms := re.FindStringSubmatch(s.Text())
		if len(ms) == 3 { // matched
			iter, e := strconv.Atoi(ms[1])
			if e != nil {
				log.Fatalf("Parsing iteration seq in %s: %v", s.Text(), e)
			}

			perplexity, e := strconv.ParseFloat(ms[2], 64)
			if e != nil {
				log.Fatalf("Parsing perplexity in %s: %v", s.Text(), e)
			}

			if iter <= prevIter {
				pts = make(plotter.XYs, 0)
			}

			pts = append(pts,
				struct{ X, Y float64 }{float64(iter), perplexity})
			prevIter = iter
		}
	}
	log.Printf("Done loading log file.")

	plotLine(pts, logFile, "Iteration", "Perplexity", imageFile)
}

func plotDocLen(docLenFile, imageFile string) {
	f, e := os.Open(docLenFile)
	r := compress_io.NewReader(f, e, path.Ext(docLenFile))
	if r == nil {
		log.Fatalf("Cannot read file %s: %v", docLenFile, e)
	}
	defer r.Close()

	log.Printf("Loading doclen file: %s ...", docLenFile)
	s := bufio.NewScanner(r)
	lens := make(plotter.Values, 0)
	for s.Scan() {
		if l, e := strconv.Atoi(s.Text()); e != nil {
			log.Fatalf("Cannot parse doclen line %s: %v", s.Text(), e)
		} else {
			lens = append(lens, float64(l))
		}
	}
	log.Printf("Done loaidng doclen file.")

	plotHist(lens, docLenFile, "Document length", "# documents", imageFile)
}

func plotVocab(vocabFile, imageFile string) {
	f, e := os.Open(vocabFile)
	r := compress_io.NewReader(f, e, path.Ext(vocabFile))
	if r == nil {
		log.Fatalf("Cannot read file %s: %v", vocabFile, e)
	}
	defer r.Close()

	log.Printf("Loading vocab file %s: ...", vocabFile)
	s := bufio.NewScanner(r)
	freq := make(plotter.Values, 0)
	for s.Scan() {
		fs := strings.Fields(s.Text())
		if len(fs) != 2 {
			log.Fatalf("Vocab file line contains not 2 fields: %s", s.Text())
		}

		if f, e := strconv.Atoi(fs[1]); e != nil {
			log.Fatalf("Cannot parse frequency in line %s: %v", s.Text(), e)
		} else {
			freq = append(freq, float64(f))
		}
	}
	log.Printf("Done laoding vocab.")

	plotHist(freq, vocabFile, "Token frequency", "# tokens", imageFile)
}

func plotLine(data plotter.XYs, title, xLabel, yLabel, imageFile string) {
	log.Printf("Plotting to %s ...", imageFile)
	p, e := plot.New()
	if e != nil {
		log.Fatalf("plot.New failed: %v", e)
	}

	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel
	p.Add(plotter.NewGrid())

	if e := plotutil.AddLinePoints(p, "", data); e != nil {
		log.Fatalf("plotutil.AddLinePoints failed: %v", e)
	}

	if e := p.Save(9, 6, imageFile); e != nil {
		log.Fatalf("Cannot save image to %s: %v", imageFile, e)
	}

	log.Printf("Done plotting log file to %s.", imageFile)
}

func plotHist(data plotter.Values, title, xLabel, yLabel, imageFile string) {
	log.Printf("Plotting to %s ...", imageFile)

	p, e := plot.New()
	if e != nil {
		log.Fatalf("plot.New failed: %v", e)
	}

	h, e := plotter.NewHist(data, 50)
	if e != nil {
		log.Fatalf("plotter.NewHist failed: %v", e)
	}
	p.Add(h)

	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel
	p.Y.Min = 1
	_, _, _, p.Y.Max = h.DataRange()
	p.Y.Scale = LogScale
	p.Y.Tick.Marker = plot.LogTicks
	p.Add(plotter.NewGrid())

	if e := p.Save(9, 6, imageFile); e != nil {
		log.Fatalf("Cannot save image to %s: %v", imageFile, e)
	}

	log.Printf("Done plotting to %s.", imageFile)
}

func LogScale(min, max, x float64) float64 {
	logMin := ln(min)
	return (ln(x) - logMin) / (ln(max) - logMin)
}

func ln(x float64) float64 {
	if x <= 0 {
		x = 0.01
	}
	return math.Log(x)
}
