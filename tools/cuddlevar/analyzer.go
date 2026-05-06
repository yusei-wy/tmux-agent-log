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
