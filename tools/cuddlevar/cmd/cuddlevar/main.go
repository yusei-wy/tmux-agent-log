package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/yusei-wy/tmux-agent-log/tools/cuddlevar"
)

func main() {
	singlechecker.Main(cuddlevar.Analyzer)
}
