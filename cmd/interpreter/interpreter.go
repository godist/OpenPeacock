package main

import (
	"flag"
	"github.com/huichen/sego"
	"github.com/wangkuiyi/phoenix/core/gibbs"
	"github.com/wangkuiyi/phoenix/core/utils"
	"html/template"
	"log"
	"net/http"
	"strings"
)

func main() {
	flagAddr := flag.String("addr", ":6061", "listening address")
	flagModel := flag.String("model", "", "model file")
	flagVocab := flag.String("vocab", "", "voabulary file")
	flagTrans := flag.String("trans", "", "vocabulary translation file")
	flagMaxWordsPerTopic := flag.Int("len", 50, "Max # tokens shown per topic")
	flagCache := flag.Int("cache", 0, "Cache in MB")
	flagSegmentor := flag.String("segmenter", "", "segmenter dictionary")
	flag.Parse()

	sgt := CreateSegmenter(*flagSegmentor)
	m, v, itr := CreateInterpreter(*flagModel, *flagVocab, *flagCache)
	if len(*flagTrans) > 0 {
		v = utils.TranslatedVocab(v,
			utils.LoadTranslationOrDie(*flagTrans))
	}
	descs := utils.DescribeTopics(m, v, *flagMaxWordsPerTopic)

	http.HandleFunc("/", MakeSafe(NewHandler(itr, sgt, descs)))
	log.Printf("Listening on %s", *flagAddr)
	if e := http.ListenAndServe(*flagAddr, nil); e != nil {
		log.Fatalf("ListenAndServe failed: %v", e)
	}
}

func CreateInterpreter(model, vocab string, cache int) (
	*gibbs.Model, *gibbs.Vocabulary, *gibbs.Interpreter) {
	m := utils.LoadModelOrDie(model)
	v := utils.LoadVocabOrDie(vocab)
	log.Printf("Smoothing model and creating interpreter ...")
	intr := gibbs.NewInterpreter(m, v, cache)
	log.Printf("Done")
	return m, v, intr
}

func CreateSegmenter(segmenter string) *sego.Segmenter {
	log.Printf("Loading segmenter %s ...", segmenter)
	sgt := new(sego.Segmenter)
	sgt.LoadDictionary(segmenter)
	log.Printf("Done")
	return sgt
}

func MakeSafe(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				log.Printf("panic: %v", e)
			}
		}()
		h(w, r)
	}
}

func NewHandler(itr *gibbs.Interpreter, sgt *sego.Segmenter,
	descs []*utils.TopicDesc) http.HandlerFunc {
	tmpl, e := template.New("interpret").Parse(kTemplate)
	if e != nil {
		log.Fatal("Cannot parse template interpret from kTemplate.")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var data []Topic

		if q := r.FormValue("q"); len(q) > 0 {
			text := make([]string, 0, len(strings.Fields(q)))
			for _, seg := range sgt.Segment([]byte(q)) {
				text = append(text, seg.Token().Text())
			}
			log.Printf("query text: %v", text)

			dist, e := itr.Interpret(text, 50, 150)
			if e != nil {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				log.Printf("Failed interpet %s: %v", text, e)
				return
			}

			numTopics := len(dist)
			if numTopics > 10 {
				numTopics = 10
			}
			data = make([]Topic, numTopics)
			for i := 0; i < numTopics; i++ {
				data[i].Weight = dist[i].Prob
				data[i].Desc = descs[dist[i].Topic]
			}
		}

		if e := tmpl.Execute(w, data); e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			log.Printf("Cannot execute HTML template.")
			return
		}
	}
}

type Topic struct {
	Weight float64
	Desc   *utils.TopicDesc
}

const (
	kTemplate = `<html>
  <head>
    <style type="text/css">
      td {font-family:Courier 10px;}
    </style>
  </head>
  <body style="background-color: #B0E2FF;">
    <form name="input" action="/" method="get" >
      <input type="textarea" name="q" size=80>
      <input type="submit" value="Interpret"></input>
    </form>
    <table>
      <thead style="border: 1px; background-color: #0198E1; color: yellow;">
        <tr>
          <td>P(topic|input)</td>
          <td>N(topic)</td>
          <td colspan=100>P(word|topic)</td>
        </tr>
      </thead>
      <tbody style="background-color: #BFEFFF; border: 1px;">
        {{range .}}
        <tr>
          <td>{{.Weight}}</td>
          {{with .Desc}}
          <td>{{.Nt}}</td>
          {{range .Tokens}}
          <td>{{.Word}}</td>
          <td>{{.Count}}</td>
          {{end}}
          {{end}}
        </tr>
      {{end}}
      </tbody>
    </table>
  </body>
</html>
`
	kMaxTopNWord = 50
)
