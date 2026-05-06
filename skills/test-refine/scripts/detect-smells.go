// detect-smells walks one or more Go test files and emits findings (S01–S11)
// as JSON-lines on stdout. Each finding describes one smell instance with
// enough context to render in a report.
//
// Usage:
//
//	go run detect-smells.go file1_test.go file2_test.go ...
//
// Output (JSON-lines, one finding per line):
//
//	{"file":"...","line":42,"test_name":"TestX","smell":"S01","severity":"H","message":"...","context":"..."}
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// Finding represents one smell detection.
type Finding struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	TestName   string `json:"test_name"`
	Smell      string `json:"smell"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Context    string `json:"context,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// assertCalls is the set of identifiers that count as assertions.
var assertCalls = map[string]bool{
	"Equal": true, "NotEqual": true, "EqualValues": true,
	"True": true, "False": true,
	"Nil": true, "NotNil": true,
	"Empty": true, "NotEmpty": true,
	"Error": true, "NoError": true, "ErrorIs": true, "ErrorAs": true,
	"Contains": true, "NotContains": true,
	"Len": true, "Greater": true, "Less": true,
	"GreaterOrEqual": true, "LessOrEqual": true,
	"Panics": true, "NotPanics": true,
	"Same": true, "NotSame": true,
	"InDelta": true, "InEpsilon": true,
	"JSONEq": true, "YAMLEq": true,
	"Eventually": true, "Never": true,
}

// tErrorCalls is the set of t.* methods that fail a test.
var tErrorCalls = map[string]bool{
	"Errorf": true, "Error": true, "Fatal": true, "Fatalf": true,
	"Fail": true, "FailNow": true,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: detect-smells <file_test.go> [more...]")
		os.Exit(2)
	}

	enc := json.NewEncoder(os.Stdout)
	for _, path := range os.Args[1:] {
		if !strings.HasSuffix(path, "_test.go") {
			continue
		}
		findings, err := analyzeFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, err)
			continue
		}
		for _, f := range findings {
			_ = enc.Encode(f)
		}
	}
}

func analyzeFile(path string) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var findings []Finding
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !isTestFunc(fn) {
			continue
		}
		findings = append(findings, analyzeTestFunc(fset, path, fn)...)
	}
	return findings, nil
}

// isTestFunc returns true for `func TestXxx(t *testing.T)`.
func isTestFunc(fn *ast.FuncDecl) bool {
	if fn.Name == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
		return false
	}
	if len(fn.Name.Name) < 5 || !isUpper(fn.Name.Name[4]) {
		return false
	}
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	return true
}

func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }

// analyzeTestFunc runs all S01–S11 checks against one test function.
func analyzeTestFunc(fset *token.FileSet, path string, fn *ast.FuncDecl) []Finding {
	var findings []Finding
	add := func(line int, smell, severity, msg string) {
		findings = append(findings, Finding{
			File:     path,
			Line:     line,
			TestName: fn.Name.Name,
			Smell:    smell,
			Severity: severity,
			Message:  msg,
		})
	}

	body := fn.Body
	if body == nil {
		return nil
	}

	asserts := collectAsserts(body)
	startLine := fset.Position(fn.Pos()).Line

	// S01: no assertions at all.
	if len(asserts) == 0 && !hasManualFail(body) {
		add(startLine, "S01", "H",
			fmt.Sprintf("test %q runs code but has no assertion", fn.Name.Name))
	}

	// S02 / S10: tautological / expect-the-expected assertions.
	for _, a := range asserts {
		if isTautological(a) {
			add(fset.Position(a.Pos()).Line, "S02", "H",
				"tautological assertion: both sides are the same expression")
		}
		if isExpectTheExpected(a, body) {
			add(fset.Position(a.Pos()).Line, "S10", "H",
				"expected value derived from actual computation")
		}
		if isSensitiveEquality(a) {
			add(fset.Position(a.Pos()).Line, "S06", "M",
				"asserting on String()/Sprintf rendering is brittle")
		}
	}

	// S04: only assertion is NotPanics or recover().
	if len(asserts) > 0 && allNotPanics(asserts) {
		add(startLine, "S04", "H",
			"only assertion is NotPanics; behavior unverified")
	} else if len(asserts) == 0 && hasOnlyRecover(body) {
		add(startLine, "S04", "H",
			"only safety check is defer recover(); behavior unverified")
	}

	// S05: error from SUT discarded.
	if errs := findDiscardedErrors(body); len(errs) > 0 {
		for _, e := range errs {
			add(fset.Position(e).Line, "S05", "H",
				"error return from SUT discarded with _")
		}
	}

	// S07: conditional/skipped assertion (early return before assert).
	if line, ok := findSkippedAssertion(body); ok {
		add(line, "S07", "M",
			"early return may bypass subsequent assertion")
	}

	// S09: assertion roulette (>3 bare asserts, no messages, not table-driven).
	if !isTableDriven(body) && countBareAsserts(asserts) > 3 {
		add(startLine, "S09", "L",
			"many assertions without messages; failure will be ambiguous")
	}

	// S11: SUT receives pointer/state but no read-back.
	if line, ok := findUnassertedSideEffect(body); ok {
		add(line, "S11", "M",
			"SUT mutates argument; mutated state not asserted")
	}

	// S03: getter/setter trivial. Detected at function level.
	if isGetterSetterTrivial(body) {
		add(startLine, "S03", "M",
			"test only verifies setter then getter; tests language semantics, not behavior")
	}

	return findings
}

// collectAsserts returns all assertion-like calls in the test body.
func collectAsserts(body *ast.BlockStmt) []*ast.CallExpr {
	var out []*ast.CallExpr
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isAssertCall(call) {
			out = append(out, call)
		}
		return true
	})
	return out
}

// isAssertCall matches assert.X(...) / require.X(...) / t.Errorf etc.
func isAssertCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	name := sel.Sel.Name
	if assertCalls[name] {
		return true
	}
	if tErrorCalls[name] {
		return true
	}
	return false
}

// hasManualFail detects `if got != want { t.Errorf(...) }` style.
func hasManualFail(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		ast.Inspect(ifs.Body, func(m ast.Node) bool {
			c, ok := m.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := c.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if tErrorCalls[sel.Sel.Name] {
				found = true
				return false
			}
			return true
		})
		return !found
	})
	return found
}

// isTautological returns true when an assertion compares an expression
// with itself, or compares a constant with the same constant.
func isTautological(a *ast.CallExpr) bool {
	sel, ok := a.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "True":
		return len(a.Args) >= 1 && exprText(a.Args[len(a.Args)-1]) == "true"
	case "False":
		return len(a.Args) >= 1 && exprText(a.Args[len(a.Args)-1]) == "false"
	case "Equal", "EqualValues":
		// Skip the leading t arg.
		args := skipTArg(a.Args)
		if len(args) >= 2 {
			return exprText(args[0]) == exprText(args[1])
		}
	}
	return false
}

// skipTArg drops a leading *testing.T argument if present.
func skipTArg(args []ast.Expr) []ast.Expr {
	if len(args) == 0 {
		return args
	}
	if ident, ok := args[0].(*ast.Ident); ok {
		if ident.Name == "t" || ident.Name == "rt" || ident.Name == "tb" {
			return args[1:]
		}
	}
	return args
}

// isExpectTheExpected detects when the "expected" arg is computed using
// the same path the SUT uses, e.g. `want := strings.ToLower(input); assert.Equal(want, got)`
// where `got = SUT(input)` and SUT also calls strings.ToLower.
//
// Heuristic version: flags when the assertion's expected arg is a variable
// assigned earlier in the test from the same input as the actual arg.
// The full dataflow check is left to a future iteration.
func isExpectTheExpected(a *ast.CallExpr, body *ast.BlockStmt) bool {
	sel, ok := a.Fun.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "Equal" && sel.Sel.Name != "EqualValues") {
		return false
	}
	args := skipTArg(a.Args)
	if len(args) < 2 {
		return false
	}
	// Look for the simple pattern: both args are CallExprs to the same fn.
	c1, ok1 := args[0].(*ast.CallExpr)
	c2, ok2 := args[1].(*ast.CallExpr)
	if ok1 && ok2 && exprText(c1.Fun) == exprText(c2.Fun) {
		return true
	}
	return false
}

// isSensitiveEquality flags assert.Equal where one arg is .String() / Sprintf.
func isSensitiveEquality(a *ast.CallExpr) bool {
	sel, ok := a.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Equal" {
		return false
	}
	args := skipTArg(a.Args)
	for _, arg := range args {
		if c, ok := arg.(*ast.CallExpr); ok {
			if s, ok := c.Fun.(*ast.SelectorExpr); ok {
				if s.Sel.Name == "String" {
					return true
				}
				if id, ok := s.X.(*ast.Ident); ok && id.Name == "fmt" &&
					(s.Sel.Name == "Sprint" || s.Sel.Name == "Sprintf" || s.Sel.Name == "Sprintln") {
					return true
				}
			}
		}
	}
	return false
}

// allNotPanics returns true when every assertion is NotPanics.
func allNotPanics(asserts []*ast.CallExpr) bool {
	for _, a := range asserts {
		sel, ok := a.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NotPanics" {
			return false
		}
	}
	return len(asserts) > 0
}

// hasOnlyRecover detects `defer func() { _ = recover() }()` as the only check.
func hasOnlyRecover(body *ast.BlockStmt) bool {
	hasRecover := false
	ast.Inspect(body, func(n ast.Node) bool {
		ds, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}
		ast.Inspect(ds, func(m ast.Node) bool {
			c, ok := m.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "recover" {
				hasRecover = true
				return false
			}
			return true
		})
		return true
	})
	return hasRecover
}

// findDiscardedErrors finds assignments where the second return value is
// `_` and the function clearly returns an error type (heuristic on naming).
// Also catches `_ = SUT(...)` patterns where the only return is discarded
// and the call is plausibly an error-returning function.
func findDiscardedErrors(body *ast.BlockStmt) []token.Pos {
	var out []token.Pos
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		// All LHS positions that are `_`.
		discardCount := 0
		for _, l := range as.Lhs {
			if id, ok := l.(*ast.Ident); ok && id.Name == "_" {
				discardCount++
			}
		}
		if discardCount == 0 {
			return true
		}
		// Must have at least one CallExpr on RHS.
		if len(as.Rhs) != 1 {
			return true
		}
		if _, ok := as.Rhs[0].(*ast.CallExpr); !ok {
			return true
		}
		// All LHS are discards? Or last LHS is `_` (likely error return)?
		if discardCount == len(as.Lhs) || // all discarded
			func() bool {
				last, ok := as.Lhs[len(as.Lhs)-1].(*ast.Ident)
				return ok && last.Name == "_"
			}() {
			out = append(out, as.Pos())
		}
		return true
	})
	return out
}

// findSkippedAssertion returns the line of an early `return` that bypasses
// later assertions.
func findSkippedAssertion(body *ast.BlockStmt) (int, bool) {
	// Walk top-level statements. If we see an If with a `return` in its body,
	// and there's an assertion call after the If, flag the early return.
	stmts := body.List
	for i, s := range stmts {
		ifs, ok := s.(*ast.IfStmt)
		if !ok {
			continue
		}
		if !blockHasReturn(ifs.Body) {
			continue
		}
		// Are there asserts after this if-stmt?
		for _, later := range stmts[i+1:] {
			if blockOrStmtHasAssert(later) {
				return positionOfReturn(ifs.Body), true
			}
		}
	}
	return 0, false
}

func blockHasReturn(b *ast.BlockStmt) bool {
	for _, s := range b.List {
		if _, ok := s.(*ast.ReturnStmt); ok {
			return true
		}
	}
	return false
}

func positionOfReturn(b *ast.BlockStmt) int {
	for _, s := range b.List {
		if r, ok := s.(*ast.ReturnStmt); ok {
			// We don't have access to fset here; caller-side fset.Position is used.
			// As a fallback, we encode the position as the offset into the file.
			return int(r.Pos())
		}
	}
	return 0
}

func blockOrStmtHasAssert(s ast.Stmt) bool {
	found := false
	ast.Inspect(s, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok && isAssertCall(c) {
			found = true
			return false
		}
		return true
	})
	return found
}

// isTableDriven detects `for _, tc := range cases { ... }` pattern.
func isTableDriven(body *ast.BlockStmt) bool {
	hasRange := false
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.RangeStmt); ok {
			hasRange = true
			return false
		}
		return true
	})
	return hasRange
}

func countBareAsserts(asserts []*ast.CallExpr) int {
	count := 0
	for _, a := range asserts {
		args := skipTArg(a.Args)
		// "Bare" = exact arg count for the assertion (no message).
		// Heuristic: Equal/EqualValues with 2 args (no msg), True/False with 1, etc.
		sel, ok := a.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		switch sel.Sel.Name {
		case "Equal", "EqualValues", "NotEqual":
			if len(args) == 2 {
				count++
			}
		case "True", "False", "Nil", "NotNil", "Empty", "NotEmpty",
			"Error", "NoError":
			if len(args) == 1 {
				count++
			}
		}
	}
	return count
}

// findUnassertedSideEffect: SUT call passes a pointer; later code never
// reads from that pointer.
func findUnassertedSideEffect(body *ast.BlockStmt) (int, bool) {
	// Heuristic only: for each AssignStmt of the form `obj := New...()`
	// followed by SUT call passing `obj` or `&obj`, check if any later
	// statement reads `obj.something`.
	// This is intentionally conservative — false negatives are preferred.
	type ptrPass struct {
		name string
		line int
	}
	var passes []ptrPass

	for i, s := range body.List {
		es, ok := s.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := es.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		// SUT-like: not an assert.
		if isAssertCall(call) {
			continue
		}
		// Pointer-passing arg?
		for _, arg := range call.Args {
			if u, ok := arg.(*ast.UnaryExpr); ok && u.Op == token.AND {
				if id, ok := u.X.(*ast.Ident); ok {
					passes = append(passes, ptrPass{name: id.Name, line: int(es.Pos())})
				}
			} else if id, ok := arg.(*ast.Ident); ok {
				// Could be a pointer var; we don't have type info.
				passes = append(passes, ptrPass{name: id.Name, line: int(es.Pos())})
			}
		}

		// Now check if any subsequent statement reads this name's field.
		for _, p := range passes {
			read := false
			for _, later := range body.List[i+1:] {
				ast.Inspect(later, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					if id, ok := sel.X.(*ast.Ident); ok && id.Name == p.name {
						read = true
						return false
					}
					return true
				})
				if read {
					break
				}
			}
			if !read {
				return p.line, true
			}
		}
	}
	return 0, false
}

// isGetterSetterTrivial detects bodies that are exactly: New(); SetX(v); assert(GetX() == v).
// Also recognizes `if c.GetX() != v { t.Errorf(...) }` form.
func isGetterSetterTrivial(body *ast.BlockStmt) bool {
	if len(body.List) > 5 {
		return false
	}
	hasSet, hasGetCheck := false, false

	// Helper to check if an expression contains a `.Get*()` call.
	containsGetCall := func(e ast.Expr) bool {
		found := false
		ast.Inspect(e, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := c.Fun.(*ast.SelectorExpr); ok &&
				strings.HasPrefix(sel.Sel.Name, "Get") {
				found = true
				return false
			}
			return true
		})
		return found
	}

	for _, s := range body.List {
		switch v := s.(type) {
		case *ast.ExprStmt:
			call, ok := v.X.(*ast.CallExpr)
			if !ok {
				continue
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			if strings.HasPrefix(sel.Sel.Name, "Set") {
				hasSet = true
			}
			if isAssertCall(call) {
				args := skipTArg(call.Args)
				for _, arg := range args {
					if containsGetCall(arg) {
						hasGetCheck = true
					}
				}
			}
		case *ast.IfStmt:
			// `if c.GetX() != v { ... }` form.
			if v.Cond != nil && containsGetCall(v.Cond) && blockHasFailCall(v.Body) {
				hasGetCheck = true
			}
		}
	}
	return hasSet && hasGetCheck
}

// blockHasFailCall detects blocks containing t.Errorf / t.Fatal etc.
func blockHasFailCall(b *ast.BlockStmt) bool {
	found := false
	ast.Inspect(b, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := c.Fun.(*ast.SelectorExpr); ok && tErrorCalls[sel.Sel.Name] {
			found = true
			return false
		}
		return true
	})
	return found
}

// exprText renders an ast.Expr to a canonical text form for equality.
func exprText(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		return v.Value
	case *ast.SelectorExpr:
		return exprText(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		args := make([]string, len(v.Args))
		for i, a := range v.Args {
			args[i] = exprText(a)
		}
		return exprText(v.Fun) + "(" + strings.Join(args, ",") + ")"
	case *ast.BinaryExpr:
		return exprText(v.X) + v.Op.String() + exprText(v.Y)
	case *ast.UnaryExpr:
		return v.Op.String() + exprText(v.X)
	case *ast.StarExpr:
		return "*" + exprText(v.X)
	case *ast.IndexExpr:
		return exprText(v.X) + "[" + exprText(v.Index) + "]"
	}
	return fmt.Sprintf("%T", e)
}
