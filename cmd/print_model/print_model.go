// print_model shows a trained model in human readable format.  It can
// output either a text file, or runs as a Web server and presents
// HTML format, depending on if -html is set.  To make the printed
// model readable, you can specify a translation file in addition to
// the vocabulary file.  For more details about translation, please
// refer to github.com/wangkuiyi/phoenix/core/utils.
package main

import (
	"flag"
	"github.com/wangkuiyi/phoenix/core/utils"
	"html/template"
	"log"
	"net/http"
	"os"
)

func main() {
	flagModel := flag.String("model", "", "The binary format model file")
	flagVocab := flag.String("vocab", "", "The vocabulary file")
	flagTrans := flag.String("trans", "", "The token translation file")
	flagMaxWordsPerTopic := flag.Int("len", 50, "Max # tokens shown per topic")
	flagHtml := flag.String("html", "", "Display HTML instead generating file")
	flag.Parse()

	v := utils.LoadVocabOrDie(*flagVocab)
	if len(*flagTrans) > 0 {
		v = utils.TranslatedVocab(v,
			utils.LoadTranslationOrDie(*flagTrans))
	}
	m := utils.LoadModelOrDie(*flagModel)

	tmpl, e := template.New("interpret").Parse(kTopicDescTemplate)
	if e != nil {
		log.Fatal("Cannot parse template interpret from kTemplate.")
	}

	if len(*flagHtml) == 0 {
		m.PrintTopics(os.Stdout, v)
	} else {
		descs := utils.DescribeTopics(m, v, *flagMaxWordsPerTopic)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if e := tmpl.Execute(w, descs); e != nil {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				log.Printf("Cannot execute HTML template: %v", e)
				return
			}
		})

		log.Printf("Listening on %s", *flagHtml)
		if e := http.ListenAndServe(*flagHtml, nil); e != nil {
			log.Fatalf("ListenAndServe failed: %v", e)
		}
	}
}

const (
	kTopicDescTemplate = `<html>
<body style="background-color: #CFEDFB">
  <table>
    <thead style="background-color: #046293; color: white;">
      <tr>
        <td>ID</td>
        <td>Frequency</td>
        <td colspan=100>Words</td>
      </tr>
    </thead>
    <tbody style="background-color: #046293; color: white;">
    {{range .}}
      <tr>
        <td>{{.Id}}</td>
        <td>{{.Nt}}</td>
        {{range .Tokens}}
          <td style="background-color: #BFEFFF;">{{.Word}}</td>
          <td style="background-color: #00A0DC; color: white;">{{.Count}}</td>
        {{end}}
      </tr>
    {{end}}
    </tbody>
  </body>
</html>
`
)
