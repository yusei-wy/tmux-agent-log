package cuddlevar_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/yusei-wy/tmux-agent-log/tools/cuddlevar"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, cuddlevar.Analyzer, "a")
}
