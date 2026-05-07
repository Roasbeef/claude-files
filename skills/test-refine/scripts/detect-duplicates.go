// detect-duplicates finds tests with structurally identical bodies (S08).
// It normalizes each test function's AST (rename locals, normalize literals)
// and groups tests by hash; groups of size > 1 are reported.
//
// Usage:
//
//	go run detect-duplicates.go file1_test.go file2_test.go ...
//
// Output: JSON-lines findings (same shape as detect-smells), one per
// duplicate test (the first member of each group is also flagged).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type Finding struct {
	File       string  `json:"file"`
	Line       int     `json:"line"`
	TestName   string  `json:"test_name"`
	Smell      string  `json:"smell"`
	Severity   string  `json:"severity"`
	Message    string  `json:"message"`
	Confidence float64 `json:"confidence,omitempty"`
	Context    string  `json:"context,omitempty"`
	Suggestion string  `json:"suggestion,omitempty"`
}

type testEntry struct {
	file string
	line int
	name string
	hash string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: detect-duplicates <file_test.go> [more...]")
		os.Exit(2)
	}

	var entries []testEntry
	fset := token.NewFileSet()
	for _, path := range os.Args[1:] {
		if !strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, err)
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !isTestFunc(fn) || fn.Body == nil {
				continue
			}
			h := hashBody(fn.Body)
			entries = append(entries, testEntry{
				file: path,
				line: fset.Position(fn.Pos()).Line,
				name: fn.Name.Name,
				hash: h,
			})
		}
	}

	// Group by hash.
	groups := map[string][]testEntry{}
	for _, e := range entries {
		groups[e.hash] = append(groups[e.hash], e)
	}

	enc := json.NewEncoder(os.Stdout)
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		// Identify peers (other test names) for the message.
		names := make([]string, 0, len(g))
		for _, e := range g {
			names = append(names, e.name)
		}
		for _, e := range g {
			peers := make([]string, 0, len(names)-1)
			for _, n := range names {
				if n != e.name {
					peers = append(peers, n)
				}
			}
			// Confidence is moderate: AST-normalized hashing is good
			// at catching true duplicates but can also catch
			// legitimate restart-variant tests that share a runner
			// (see DEF1). Until DEF1 lands, surface as low-confidence.
			_ = enc.Encode(Finding{
				File:     e.file,
				Line:     e.line,
				TestName: e.name,
				Smell:    "S08",
				Severity: "M",
				Message: fmt.Sprintf("structurally duplicate test of: %s",
					strings.Join(peers, ", ")),
				Confidence: 0.6,
				Suggestion: "consolidate into a table-driven test, or delete duplicates",
			})
		}
	}
}

func isTestFunc(fn *ast.FuncDecl) bool {
	if fn.Name == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
		return false
	}
	if len(fn.Name.Name) < 5 {
		return false
	}
	c := fn.Name.Name[4]
	return c >= 'A' && c <= 'Z'
}

// hashBody returns a hash of a normalized rendering of the function body.
// Normalization: rename locals to v0, v1, ...; collapse string and numeric
// literals into placeholders; strip comments.
func hashBody(body *ast.BlockStmt) string {
	r := &renderer{
		locals:   map[string]string{},
		literals: map[string]string{},
	}
	out := r.renderStmt(body)
	sum := sha256.Sum256([]byte(out))
	return hex.EncodeToString(sum[:])
}

type renderer struct {
	locals   map[string]string
	literals map[string]string
	li       int
	li2      int
}

func (r *renderer) renderStmt(s ast.Stmt) string {
	switch v := s.(type) {
	case *ast.BlockStmt:
		parts := []string{"{"}
		for _, st := range v.List {
			parts = append(parts, r.renderStmt(st))
		}
		parts = append(parts, "}")
		return strings.Join(parts, ";")

	case *ast.AssignStmt:
		// Normalize LHS identifiers.
		lhs := make([]string, len(v.Lhs))
		for i, e := range v.Lhs {
			lhs[i] = r.renderExpr(e)
		}
		rhs := make([]string, len(v.Rhs))
		for i, e := range v.Rhs {
			rhs[i] = r.renderExpr(e)
		}
		return strings.Join(lhs, ",") + v.Tok.String() + strings.Join(rhs, ",")

	case *ast.ExprStmt:
		return r.renderExpr(v.X)

	case *ast.ReturnStmt:
		parts := []string{"return"}
		for _, e := range v.Results {
			parts = append(parts, r.renderExpr(e))
		}
		return strings.Join(parts, " ")

	case *ast.IfStmt:
		out := "if " + r.renderExpr(v.Cond) + r.renderStmt(v.Body)
		if v.Else != nil {
			out += "else" + r.renderStmt(v.Else)
		}
		return out

	case *ast.ForStmt:
		out := "for"
		if v.Init != nil {
			out += " " + r.renderStmt(v.Init)
		}
		if v.Cond != nil {
			out += ";" + r.renderExpr(v.Cond)
		}
		if v.Post != nil {
			out += ";" + r.renderStmt(v.Post)
		}
		out += r.renderStmt(v.Body)
		return out

	case *ast.RangeStmt:
		out := "for " + r.renderExpr(v.Key) + ","
		if v.Value != nil {
			out += r.renderExpr(v.Value)
		}
		out += " range " + r.renderExpr(v.X) + r.renderStmt(v.Body)
		return out

	case *ast.DeclStmt:
		return "<decl>"

	case *ast.IncDecStmt:
		return r.renderExpr(v.X) + v.Tok.String()

	case *ast.DeferStmt:
		return "defer " + r.renderExpr(v.Call)

	case *ast.GoStmt:
		return "go " + r.renderExpr(v.Call)

	case *ast.SwitchStmt:
		out := "switch"
		if v.Tag != nil {
			out += " " + r.renderExpr(v.Tag)
		}
		out += r.renderStmt(v.Body)
		return out

	case *ast.CaseClause:
		out := "case"
		for _, e := range v.List {
			out += " " + r.renderExpr(e)
		}
		out += ":"
		for _, st := range v.Body {
			out += r.renderStmt(st) + ";"
		}
		return out

	case *ast.SelectStmt:
		return "select" + r.renderStmt(v.Body)

	case *ast.SendStmt:
		return r.renderExpr(v.Chan) + "<-" + r.renderExpr(v.Value)

	case *ast.LabeledStmt:
		return r.renderStmt(v.Stmt)

	case *ast.BranchStmt:
		return v.Tok.String()
	}
	return fmt.Sprintf("<%T>", s)
}

func (r *renderer) renderExpr(e ast.Expr) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case *ast.Ident:
		// Normalize local-looking identifiers.
		if isLikelyLocal(v.Name) {
			if alias, ok := r.locals[v.Name]; ok {
				return alias
			}
			alias := fmt.Sprintf("v%d", r.li)
			r.li++
			r.locals[v.Name] = alias
			return alias
		}
		return v.Name

	case *ast.BasicLit:
		// Collapse literal kinds, but keep distinct values identifiable
		// up to canonical placeholders: <int>, <str>, <float>, <imag>, <char>.
		switch v.Kind {
		case token.INT:
			return "<int>"
		case token.FLOAT:
			return "<float>"
		case token.IMAG:
			return "<imag>"
		case token.STRING:
			return "<str>"
		case token.CHAR:
			return "<char>"
		}
		return "<lit>"

	case *ast.SelectorExpr:
		return r.renderExpr(v.X) + "." + v.Sel.Name

	case *ast.CallExpr:
		args := make([]string, len(v.Args))
		for i, a := range v.Args {
			args[i] = r.renderExpr(a)
		}
		return r.renderExpr(v.Fun) + "(" + strings.Join(args, ",") + ")"

	case *ast.BinaryExpr:
		return r.renderExpr(v.X) + v.Op.String() + r.renderExpr(v.Y)

	case *ast.UnaryExpr:
		return v.Op.String() + r.renderExpr(v.X)

	case *ast.ParenExpr:
		return "(" + r.renderExpr(v.X) + ")"

	case *ast.StarExpr:
		return "*" + r.renderExpr(v.X)

	case *ast.IndexExpr:
		return r.renderExpr(v.X) + "[" + r.renderExpr(v.Index) + "]"

	case *ast.SliceExpr:
		return r.renderExpr(v.X) + "[" + r.renderExpr(v.Low) + ":" + r.renderExpr(v.High) + "]"

	case *ast.CompositeLit:
		elts := make([]string, len(v.Elts))
		for i, el := range v.Elts {
			elts[i] = r.renderExpr(el)
		}
		return r.renderExpr(v.Type) + "{" + strings.Join(elts, ",") + "}"

	case *ast.KeyValueExpr:
		return r.renderExpr(v.Key) + ":" + r.renderExpr(v.Value)

	case *ast.FuncLit:
		return "func" + r.renderStmt(v.Body)

	case *ast.ArrayType:
		return "[]" + r.renderExpr(v.Elt)

	case *ast.MapType:
		return "map[" + r.renderExpr(v.Key) + "]" + r.renderExpr(v.Value)

	case *ast.StructType:
		return "struct{}"

	case *ast.InterfaceType:
		return "interface{}"

	case *ast.ChanType:
		return "chan " + r.renderExpr(v.Value)

	case *ast.TypeAssertExpr:
		return r.renderExpr(v.X) + ".(" + r.renderExpr(v.Type) + ")"

	case *ast.Ellipsis:
		return "..." + r.renderExpr(v.Elt)
	}
	return fmt.Sprintf("<%T>", e)
}

// isLikelyLocal returns true for short, lowercase-starting identifiers
// that look like locals rather than package-level names.
func isLikelyLocal(name string) bool {
	if name == "" {
		return false
	}
	if name == "_" || name == "true" || name == "false" || name == "nil" {
		return false
	}
	first := name[0]
	if first < 'a' || first > 'z' {
		return false
	}
	// Don't normalize obvious package idents we want to keep.
	switch name {
	case "t", "rt", "tb", "ctx", "err":
		// These are commonly used and structural — keep as-is so they
		// drive duplicate detection.
		return false
	}
	return true
}
