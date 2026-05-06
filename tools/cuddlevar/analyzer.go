package cuddlevar

import "golang.org/x/tools/go/analysis"

var Analyzer = &analysis.Analyzer{
	Name: "cuddlevar",
	Doc:  "removes unnecessary blank lines between assignment and block using the assigned variable",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	return nil, nil
}
