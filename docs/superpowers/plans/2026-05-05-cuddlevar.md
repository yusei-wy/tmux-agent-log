# cuddlevar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 代入した変数が直後のブロック文で使われている場合、間の不要な空行を除去する autofix 付き golangci-lint カスタム Analyzer を作る

**Architecture:** `go/analysis.Analyzer` として実装し、golangci-lint module plugin として統合する。Go workspace で本体モジュールと分離。standalone 実行 (`go vet -vettool`) もサポート。

**Tech Stack:** Go 1.26, `golang.org/x/tools/go/analysis`, `github.com/golangci/plugin-module-register`, golangci-lint v2.12.1

---

### Task 1: Module scaffolding

**Files:**
- Create: `go.work`
- Create: `tools/cuddlevar/go.mod`
- Create: `tools/cuddlevar/analyzer.go` (skeleton)

- [ ] **Step 1: Create Go workspace file**

```go
// go.work
go 1.26

use (
	.
	./tools/cuddlevar
)
```

- [ ] **Step 2: Create cuddlevar module**

```bash
mkdir -p tools/cuddlevar
cd tools/cuddlevar
go mod init github.com/yusei-wy/tmux-agent-log/tools/cuddlevar
```

- [ ] **Step 3: Write skeleton analyzer**

```go
// tools/cuddlevar/analyzer.go
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
```

- [ ] **Step 4: Add dependencies and tidy**

```bash
cd tools/cuddlevar
go get golang.org/x/tools/go/analysis
go mod tidy
```

- [ ] **Step 5: Verify build**

Run: `cd tools/cuddlevar && go build ./...`
Expected: success, no output

- [ ] **Step 6: Commit**

```bash
git add go.work tools/cuddlevar/
git commit -m "feat(cuddlevar): scaffold module with skeleton analyzer"
```

---

### Task 2: Test cases and test runner

**Files:**
- Create: `tools/cuddlevar/testdata/src/a/a.go`
- Create: `tools/cuddlevar/analyzer_test.go`

- [ ] **Step 1: Write test input with want comments**

```go
// tools/cuddlevar/testdata/src/a/a.go
package a

// --- should detect ---

func assignBeforeIf() {
	x := 1

	if x > 0 { // want `unnecessary blank line before block using x`
		println(x)
	}
}

func assignBeforeFor() {
	items := []int{1, 2, 3}

	for _, item := range items { // want `unnecessary blank line before block using items`
		println(item)
	}
}

func assignBeforeSwitch() {
	mode := "json"

	switch mode { // want `unnecessary blank line before block using mode`
	case "json":
		println("json")
	}
}

func assignBeforeGoFunc() {
	ch := make(chan int)

	go func() { // want `unnecessary blank line before block using ch`
		ch <- 1
	}()
}

func assignBeforeDeferFunc() {
	cleanup := func() {}

	defer func() { // want `unnecessary blank line before block using cleanup`
		cleanup()
	}()
}

func varDecl() {
	var x int

	if x > 0 { // want `unnecessary blank line before block using x`
		println(x)
	}
}

func multiAssign() {
	x, y := 1, 2

	if y > 0 { // want `unnecessary blank line before block using y`
		println(y)
	}
	_ = x
}

func multipleBlankLines() {
	x := 1


	if x > 0 { // want `unnecessary blank line before block using x`
		println(x)
	}
}

// --- should NOT detect ---

func unusedInBlock() {
	x := 1

	if true {
		println("hello")
	}
	_ = x
}

func alreadyCuddled() {
	x := 1
	if x > 0 {
		println(x)
	}
}

func commentBetween() {
	x := 1
	// this is intentional
	if x > 0 {
		println(x)
	}
}

func notAssignment() {
	println("hello")

	if true {
		println("world")
	}
}

func goPlainCall() {
	ch := make(chan int)

	go println(ch)
}

func blankIdentifier() {
	_ = 1

	if true {
		println("hello")
	}
}
```

- [ ] **Step 2: Write test runner**

```go
// tools/cuddlevar/analyzer_test.go
package cuddlevar_test

import (
	"testing"

	"github.com/yusei-wy/tmux-agent-log/tools/cuddlevar"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, cuddlevar.Analyzer, "a")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd tools/cuddlevar && go test ./...`
Expected: FAIL — `want` comments expect diagnostics but skeleton analyzer reports none

- [ ] **Step 4: Commit**

```bash
git add tools/cuddlevar/testdata/ tools/cuddlevar/analyzer_test.go
git commit -m "test(cuddlevar): add test cases for blank line detection"
```

---

### Task 3: Analyzer implementation (diagnostics only)

**Files:**
- Modify: `tools/cuddlevar/analyzer.go`

- [ ] **Step 1: Implement the full analyzer**

```go
// tools/cuddlevar/analyzer.go
package cuddlevar

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "cuddlevar",
	Doc:  "removes unnecessary blank lines between assignment and block using the assigned variable",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			block, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			for i := 0; i < len(block.List)-1; i++ {
				checkPair(pass, file, block.List[i], block.List[i+1])
			}
			return true
		})
	}
	return nil, nil
}

func checkPair(pass *analysis.Pass, file *ast.File, prev, next ast.Stmt) {
	names := assignedNames(prev)
	if len(names) == 0 {
		return
	}
	if !isBlockStmt(next) {
		return
	}

	tokenFile := pass.Fset.File(prev.End())
	prevEndLine := tokenFile.Line(prev.End())
	nextStartLine := tokenFile.Line(next.Pos())
	if nextStartLine-prevEndLine <= 1 {
		return
	}

	if hasCommentBetween(file, prev.End(), next.Pos()) {
		return
	}

	used := firstUsedName(names, next)
	if used == "" {
		return
	}

	pass.Report(analysis.Diagnostic{
		Pos:     next.Pos(),
		Message: fmt.Sprintf("unnecessary blank line before block using %s", used),
	})
}

func assignedNames(stmt ast.Stmt) []string {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		var names []string
		for _, lhs := range s.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok && ident.Name != "_" {
				names = append(names, ident.Name)
			}
		}
		return names
	case *ast.DeclStmt:
		gd, ok := s.Decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			return nil
		}
		var names []string
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, n := range vs.Names {
				if n.Name != "_" {
					names = append(names, n.Name)
				}
			}
		}
		return names
	default:
		return nil
	}
}

func isBlockStmt(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt,
		*ast.SwitchStmt, *ast.TypeSwitchStmt:
		return true
	case *ast.GoStmt:
		_, ok := s.Call.Fun.(*ast.FuncLit)
		return ok
	case *ast.DeferStmt:
		_, ok := s.Call.Fun.(*ast.FuncLit)
		return ok
	default:
		return false
	}
}

func hasCommentBetween(file *ast.File, start, end token.Pos) bool {
	for _, cg := range file.Comments {
		if cg.Pos() > start && cg.End() < end {
			return true
		}
	}
	return false
}

func firstUsedName(names []string, node ast.Node) string {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	var found string
	ast.Inspect(node, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if ok && nameSet[ident.Name] {
			found = ident.Name
			return false
		}
		return true
	})
	return found
}
```

- [ ] **Step 2: Run test to verify diagnostics pass**

Run: `cd tools/cuddlevar && go test -v ./...`
Expected: PASS — all `want` comments match, no-detect cases report nothing

- [ ] **Step 3: Commit**

```bash
git add tools/cuddlevar/analyzer.go
git commit -m "feat(cuddlevar): implement blank line detection for cuddled variables"
```

---

### Task 4: Autofix (SuggestedFixes)

**Files:**
- Modify: `tools/cuddlevar/analyzer.go` (add SuggestedFixes to Diagnostic)
- Create: `tools/cuddlevar/testdata/src/a/a.go.golden`

- [ ] **Step 1: Add SuggestedFixes to the Diagnostic**

`analyzer.go` の `checkPair` 関数内、`pass.Report` を修正:

```go
	editStart := tokenFile.LineStart(prevEndLine + 1)
	editEnd := tokenFile.LineStart(nextStartLine)

	pass.Report(analysis.Diagnostic{
		Pos:     next.Pos(),
		Message: fmt.Sprintf("unnecessary blank line before block using %s", used),
		SuggestedFixes: []analysis.SuggestedFix{
			{
				Message: fmt.Sprintf("Remove blank line before block using %s", used),
				TextEdits: []analysis.TextEdit{
					{
						Pos:     editStart,
						End:     editEnd,
						NewText: nil,
					},
				},
			},
		},
	})
```

`editStart` は代入文の次の行の先頭、`editEnd` はブロック文の行の先頭。この範囲（空行のみ）を削除する。

- [ ] **Step 2: Write golden file**

`tools/cuddlevar/testdata/src/a/a.go.golden` — 検出対象箇所の空行を除去した期待出力。検出対象外の箇所は元のまま。`// want` コメントもそのまま残す（analysistest が処理する）。

```go
// tools/cuddlevar/testdata/src/a/a.go.golden
package a

// --- should detect ---

func assignBeforeIf() {
	x := 1
	if x > 0 { // want `unnecessary blank line before block using x`
		println(x)
	}
}

func assignBeforeFor() {
	items := []int{1, 2, 3}
	for _, item := range items { // want `unnecessary blank line before block using items`
		println(item)
	}
}

func assignBeforeSwitch() {
	mode := "json"
	switch mode { // want `unnecessary blank line before block using mode`
	case "json":
		println("json")
	}
}

func assignBeforeGoFunc() {
	ch := make(chan int)
	go func() { // want `unnecessary blank line before block using ch`
		ch <- 1
	}()
}

func assignBeforeDeferFunc() {
	cleanup := func() {}
	defer func() { // want `unnecessary blank line before block using cleanup`
		cleanup()
	}()
}

func varDecl() {
	var x int
	if x > 0 { // want `unnecessary blank line before block using x`
		println(x)
	}
}

func multiAssign() {
	x, y := 1, 2
	if y > 0 { // want `unnecessary blank line before block using y`
		println(y)
	}
	_ = x
}

func multipleBlankLines() {
	x := 1
	if x > 0 { // want `unnecessary blank line before block using x`
		println(x)
	}
}

// --- should NOT detect ---

func unusedInBlock() {
	x := 1

	if true {
		println("hello")
	}
	_ = x
}

func alreadyCuddled() {
	x := 1
	if x > 0 {
		println(x)
	}
}

func commentBetween() {
	x := 1
	// this is intentional
	if x > 0 {
		println(x)
	}
}

func notAssignment() {
	println("hello")

	if true {
		println("world")
	}
}

func goPlainCall() {
	ch := make(chan int)

	go println(ch)
}

func blankIdentifier() {
	_ = 1

	if true {
		println("hello")
	}
}
```

- [ ] **Step 3: Run test to verify fix output matches golden**

Run: `cd tools/cuddlevar && go test -v ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add tools/cuddlevar/
git commit -m "feat(cuddlevar): add autofix via SuggestedFixes"
```

---

### Task 5: golangci-lint plugin adapter

**Files:**
- Create: `tools/cuddlevar/plugin.go`

- [ ] **Step 1: Write plugin adapter**

```go
// tools/cuddlevar/plugin.go
package cuddlevar

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("cuddlevar", newPlugin)
}

func newPlugin(any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (*plugin) BuildAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{Analyzer}
}

func (*plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}
```

- [ ] **Step 2: Add dependency and tidy**

```bash
cd tools/cuddlevar
go get github.com/golangci/plugin-module-register
go mod tidy
```

- [ ] **Step 3: Verify build**

Run: `cd tools/cuddlevar && go build ./...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add tools/cuddlevar/
git commit -m "feat(cuddlevar): add golangci-lint module plugin adapter"
```

---

### Task 6: Standalone command

**Files:**
- Create: `tools/cuddlevar/cmd/cuddlevar/main.go`

- [ ] **Step 1: Write standalone entry point**

```go
// tools/cuddlevar/cmd/cuddlevar/main.go
package main

import (
	"github.com/yusei-wy/tmux-agent-log/tools/cuddlevar"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(cuddlevar.Analyzer)
}
```

- [ ] **Step 2: Verify build**

Run: `cd tools/cuddlevar && go build -o /dev/null ./cmd/cuddlevar/`
Expected: success

- [ ] **Step 3: Smoke test on real code**

```bash
cd tools/cuddlevar
go build -o /tmp/cuddlevar ./cmd/cuddlevar/
cd ../..
go vet -vettool=/tmp/cuddlevar ./internal/cli/
```

Expected: 指摘があればファイル名と行番号が表示される（clear.go 等）

- [ ] **Step 4: Commit**

```bash
git add tools/cuddlevar/cmd/
git commit -m "feat(cuddlevar): add standalone go vet -vettool command"
```

---

### Task 7: golangci-lint integration

**Files:**
- Create: `.custom-gcl.yml`
- Modify: `.golangci.yml`
- Modify: `.mise.toml`
- Modify: `.gitignore`

- [ ] **Step 1: Create .custom-gcl.yml**

```yaml
# .custom-gcl.yml
version: v2.12.1
plugins:
  - module: github.com/yusei-wy/tmux-agent-log/tools/cuddlevar
    path: tools/cuddlevar
```

- [ ] **Step 2: Build custom binary**

```bash
golangci-lint custom --destination tools/bin --name custom-gcl
```

Expected: `tools/bin/custom-gcl` が生成される

- [ ] **Step 3: Add tools/bin/ to .gitignore**

`.gitignore` に追加:

```
tools/bin/
```

- [ ] **Step 4: Enable cuddlevar in .golangci.yml**

`linters.enable` に `cuddlevar` を追加し、`linters.settings.custom` に定義を追加:

```yaml
linters:
  enable:
    # ... existing ...
    - cuddlevar
  settings:
    # ... existing ...
    custom:
      cuddlevar:
        type: module
```

- [ ] **Step 5: Update .mise.toml lint tasks**

```toml
[tasks."lint:build"]
description = "Build custom golangci-lint with cuddlevar"
run = "golangci-lint custom --destination tools/bin --name custom-gcl"

[tasks.lint]
description = "Run golangci-lint (custom build)"
run = "tools/bin/custom-gcl run"

[tasks."lint:fix"]
description = "Run golangci-lint --fix (custom build)"
run = "tools/bin/custom-gcl run --fix"
```

- [ ] **Step 6: Verify config**

```bash
mise run lint:build
tools/bin/custom-gcl config verify
```

Expected: エラーなし

- [ ] **Step 7: Commit**

```bash
git add .custom-gcl.yml .golangci.yml .mise.toml .gitignore
git commit -m "build: integrate cuddlevar into golangci-lint via custom binary"
```

---

### Task 8: Run on codebase and fix

- [ ] **Step 1: Run lint to see cuddlevar findings**

```bash
mise run lint 2>&1 | grep cuddlevar
```

Expected: clear.go 等で指摘が出る

- [ ] **Step 2: Auto-fix all findings**

```bash
mise run lint:fix
```

- [ ] **Step 3: Verify zero issues**

```bash
mise run lint
```

Expected: 0 issues

- [ ] **Step 4: Run tests**

```bash
mise run test
```

Expected: all pass

- [ ] **Step 5: Commit fixed files**

```bash
git add -A
git commit -m "style: apply cuddlevar auto-fix across codebase"
```
