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
	FixKind    string  `json:"fix_kind,omitempty"`
	Context    string  `json:"context,omitempty"`
	Suggestion string  `json:"suggestion,omitempty"`
}

type testEntry struct {
	file string
	line int
	name string
	hash string
	// delegation describes the test's body when it is a single-call
	// delegation to a runner. nil otherwise.
	delegation *delegationShape
}

// delegationShape captures the runner-call signature: the callee name
// and a normalized arg list. Two tests that share a runner but differ
// only in one constant-typed argument are restart-variant siblings,
// not real duplicates.
type delegationShape struct {
	callee string
	args   []string // canonical arg renderings; "<lit>" for literals
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
				file:       path,
				line:       fset.Position(fn.Pos()).Line,
				name:       fn.Name.Name,
				hash:       h,
				delegation: delegationOf(fn.Body),
			})
		}
	}

	// Group entries that hash identically. These hash to the same
	// AST modulo locals and literal values, so they're either real
	// duplicates or restart-variant siblings (single-call delegations
	// to the same runner with different constant args).
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

		// Restart-variant detection: every group member is a single
		// delegation to the same runner. In that case demote from
		// duplicate-removal candidate to a softer "consider table-
		// driving" suggestion.
		shape, allDelegate := commonDelegation(g)
		smell := "S08"
		severity := "M"
		conf := 0.6
		fixKind := "auto"
		message := func(peers []string) string {
			return fmt.Sprintf("structurally duplicate test of: %s",
				strings.Join(peers, ", "))
		}
		suggestion := "consolidate into a table-driven test, or delete duplicates"
		if allDelegate && shape != nil {
			smell = "S08-VARIANT"
			severity = "L"
			conf = 0.4
			fixKind = "manual" // demote — needs human judgment.
			message = func(peers []string) string {
				return fmt.Sprintf(
					"restart-variant sibling of: %s (single call to %s, differs only in arguments)",
					strings.Join(peers, ", "), shape.callee)
			}
			suggestion = "if the variants share intent, consider a table-driven form; otherwise leave as-is"
		}

		for _, e := range g {
			peers := make([]string, 0, len(names)-1)
			for _, n := range names {
				if n != e.name {
					peers = append(peers, n)
				}
			}
			_ = enc.Encode(Finding{
				File:       e.file,
				Line:       e.line,
				TestName:   e.name,
				Smell:      smell,
				Severity:   severity,
				Message:    message(peers),
				Confidence: conf,
				FixKind:    fixKind,
				Suggestion: suggestion,
			})
		}
	}
}

// delegationOf returns a delegationShape iff the body is a single
// ExprStmt CallExpr with a resolvable callee — i.e. the test runs by
// delegating to one helper.
func delegationOf(body *ast.BlockStmt) *delegationShape {
	if body == nil || len(body.List) != 1 {
		return nil
	}
	es, ok := body.List[0].(*ast.ExprStmt)
	if !ok {
		return nil
	}
	call, ok := es.X.(*ast.CallExpr)
	if !ok {
		return nil
	}
	var callee string
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		callee = fn.Name
	case *ast.SelectorExpr:
		callee = fn.Sel.Name
	default:
		return nil
	}
	args := make([]string, len(call.Args))
	for i, a := range call.Args {
		args[i] = canonicalArg(a)
	}
	return &delegationShape{callee: callee, args: args}
}

// canonicalArg renders an argument in a form that distinguishes
// "literal" from named identifiers. Used to detect "differs only in
// constant arguments" between sibling tests.
func canonicalArg(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.BasicLit:
		return "<lit>"
	case *ast.Ident:
		if v.Name == "true" || v.Name == "false" || v.Name == "nil" {
			return "<const>"
		}
		// Named constants (often iota-typed enums) are exactly the
		// thing restart-variant siblings differ in. Treat as <const>
		// so they collapse together.
		if isUpperFirst(v.Name) {
			return "<const>"
		}
		return v.Name
	case *ast.SelectorExpr:
		return canonicalArg(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		// E.g., context.Background(); preserve callee shape.
		if id, ok := v.Fun.(*ast.Ident); ok {
			return id.Name + "()"
		}
		if sel, ok := v.Fun.(*ast.SelectorExpr); ok {
			return canonicalArg(sel.X) + "." + sel.Sel.Name + "()"
		}
		return "<call>"
	}
	return "<expr>"
}

func isUpperFirst(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return c >= 'A' && c <= 'Z'
}

// commonDelegation returns the shared delegation shape iff every entry
// in the group is a delegation to the same callee. The returned shape
// is one canonical instance; callers that need per-entry args should
// re-derive them.
func commonDelegation(group []testEntry) (*delegationShape, bool) {
	if len(group) == 0 {
		return nil, false
	}
	first := group[0].delegation
	if first == nil {
		return nil, false
	}
	for _, e := range group[1:] {
		if e.delegation == nil || e.delegation.callee != first.callee {
			return nil, false
		}
		if len(e.delegation.args) != len(first.args) {
			return nil, false
		}
	}
	return first, true
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
		// Render the callee specially: a bare ident in callee position
		// is almost always a function (package-level or imported), not
		// a local. Aliasing it would collapse calls to *different*
		// runners into the same hash, producing false-positive S08
		// findings on tests that delegate to distinct helpers. Keep
		// the literal name.
		var callee string
		switch fn := v.Fun.(type) {
		case *ast.Ident:
			callee = fn.Name
		default:
			callee = r.renderExpr(v.Fun)
		}
		return callee + "(" + strings.Join(args, ",") + ")"

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
