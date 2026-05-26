package main

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
)

type TickerSummaryView struct {
	Dividends float64
}

func main() {
	tmpl := `<td class="px-6 py-4 font-bold {{if gt .Dividends 0.0}}text-teal-600 dark:text-teal-400{{else}}text-gray-400 dark:text-gray-500{{end}}">
                            {{if gt .Dividends 0.0}}+{{printf "%.2f" .Dividends}}{{else}}—{{end}}
                        </td>`
	t, err := template.New("test").Parse(tmpl)
	if err != nil {
		panic(err)
	}
	data := TickerSummaryView{Dividends: 0.0}
	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		panic(err)
	}
	fmt.Println("SUCCESS")
}
