// Separate module: keeps the kagome tokenizer + IPADIC dictionary out of the
// main polyglot module's dependency graph. This tool is run offline to
// regenerate content/ja/frequency.tsv; see README.md.
module github.com/sebastiancaraballo/polyglot/tools/genfreq

go 1.26.1

require (
	github.com/ikawaha/kagome-dict/ipa v1.2.6
	github.com/ikawaha/kagome/v2 v2.11.0
)

require github.com/ikawaha/kagome-dict v1.1.7 // indirect
