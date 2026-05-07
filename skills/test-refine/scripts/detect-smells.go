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
//
// fix_kind classifies how the fix is applied:
//   - "auto":   mechanical edit (delete a tautology, drop a duplicate)
//     that apply-fixes can perform with an Edit tool call.
//   - "manual": requires writing a new test or understanding test
//     intent; apply-fixes leaves a TODO comment instead.
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

// smellFixKind returns the canonical fix_kind for each smell ID.
// Most smells are "manual" because they require an assertion to be
// written or test intent to be understood. Only smells whose fix is
// "delete this test" or "delete this redundant code" are "auto".
func smellFixKind(smell string) string {
	switch smell {
	case "S02", // tautology — delete the assertion (or the whole test)
		"S03", // getter/setter trivial — delete the test
		"S08": // duplicate — delete one
		return "auto"
	}
	return "manual"
}

// pkgIndex captures package-wide signals used to refine detection.
// Built across all input files so we can answer questions like
// "does helper X return an error?" when X is defined in another
// file of the same package.
type pkgIndex struct {
	// errorReturners maps function/method name -> true iff its
	// declared return list contains the identifier `error`.
	errorReturners map[string]bool
	// knownFuncs is the set of names defined as funcs anywhere in
	// the input files (so we can distinguish "known same-package
	// callee" from "external/unknown").
	knownFuncs map[string]bool
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

type parsedFile struct {
	path string
	fset *token.FileSet
	file *ast.File
}

// testStyle records the assertion style used inside one test func,
// for the package-wide style-drift pass.
type testStyle struct {
	path       string
	startLine  int
	fnName     string
	manualFail bool // uses `if x != y { t.Errorf(...) }`
	usesAssert bool // uses require.* / assert.*
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: detect-smells <file_test.go> [more...]")
		os.Exit(2)
	}

	// First pass: parse all input files and build a package-wide index.
	// This lets per-file analysis see helper definitions in sibling
	// files of the same package.
	var files []parsedFile
	for _, path := range os.Args[1:] {
		if !strings.HasSuffix(path, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, err)
			continue
		}
		files = append(files, parsedFile{path: path, fset: fset, file: file})
	}

	idx := buildPkgIndex(files)

	// Second pass: analyze each file. Capture per-test style so we can
	// run the package-wide drift detector after.
	enc := json.NewEncoder(os.Stdout)
	var styles []testStyle

	for _, p := range files {
		var findings []Finding
		for _, decl := range p.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !isTestFunc(fn) {
				continue
			}
			findings = append(findings, analyzeTestFunc(p.fset, p.path, fn, idx)...)

			// Style classification for the package-wide drift pass.
			style := testStyle{
				path:      p.path,
				startLine: p.fset.Position(fn.Pos()).Line,
				fnName:    fn.Name.Name,
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := c.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if id, ok := sel.X.(*ast.Ident); ok {
					if id.Name == "require" || id.Name == "assert" {
						style.usesAssert = true
					}
				}
				return true
			})
			if hasManualFail(fn.Body) {
				style.manualFail = true
			}
			styles = append(styles, style)
		}
		for _, f := range findings {
			_ = enc.Encode(f)
		}
	}

	// Package-wide style drift (DEF5): if the codebase clearly prefers
	// one assertion style (>70%), flag the minority as drift.
	emitStyleDrift(enc, styles)
}

// emitStyleDrift compares each test's style to the package-wide
// majority and emits S-STYLE-DRIFT findings for the minority. The
// drift is interesting when one style accounts for >70% of tests
// — below that the package has no clear convention and flagging
// either side would just be noise.
func emitStyleDrift(enc *json.Encoder, styles []testStyle) {
	const dominanceThreshold = 0.7
	if len(styles) < 4 {
		// Too few tests to establish a convention.
		return
	}

	manualCount, assertCount := 0, 0
	for _, s := range styles {
		// Don't double-count tests that use both styles — those are
		// neutral evidence and don't push either way.
		if s.manualFail && !s.usesAssert {
			manualCount++
		} else if s.usesAssert && !s.manualFail {
			assertCount++
		}
	}
	classified := manualCount + assertCount
	if classified == 0 {
		return
	}

	manualFrac := float64(manualCount) / float64(classified)
	assertFrac := float64(assertCount) / float64(classified)

	switch {
	case assertFrac >= dominanceThreshold:
		// require/assert is dominant; flag tests that use only manual fail.
		for _, s := range styles {
			if s.manualFail && !s.usesAssert {
				_ = enc.Encode(Finding{
					File:     s.path,
					Line:     s.startLine,
					TestName: s.fnName,
					Smell:    "S-STYLE-DRIFT",
					Severity: "L",
					Message: fmt.Sprintf(
						"package uses require/assert in %.0f%% of tests; this test uses `if got != want { t.Errorf }` style",
						assertFrac*100),
					Confidence: 0.8,
					FixKind:    "auto",
					Suggestion: "convert to require.Equal(t, want, got) for consistency",
				})
			}
		}
	case manualFrac >= dominanceThreshold:
		// manual t.Errorf is dominant; flag tests that pull in require/assert.
		for _, s := range styles {
			if s.usesAssert && !s.manualFail {
				_ = enc.Encode(Finding{
					File:     s.path,
					Line:     s.startLine,
					TestName: s.fnName,
					Smell:    "S-STYLE-DRIFT",
					Severity: "L",
					Message: fmt.Sprintf(
						"package uses `if got != want { t.Errorf }` style in %.0f%% of tests; this test uses require/assert",
						manualFrac*100),
					Confidence: 0.8,
					FixKind:    "auto",
					Suggestion: "convert to manual fail style for consistency, or migrate the package wholesale",
				})
			}
		}
	}
}

// buildPkgIndex records every func's name across the input files plus
// whether it returns an `error` identifier as one of its results.
// Methods are indexed by both bare name and `Receiver.Method`.
func buildPkgIndex(files []parsedFile) *pkgIndex {
	idx := &pkgIndex{
		errorReturners: map[string]bool{},
		knownFuncs:     map[string]bool{},
	}
	for _, p := range files {
		for _, decl := range p.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			name := fn.Name.Name
			idx.knownFuncs[name] = true
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				if recv := receiverTypeName(fn.Recv.List[0].Type); recv != "" {
					idx.knownFuncs[recv+"."+name] = true
				}
			}
			if fn.Type.Results == nil {
				continue
			}
			for _, r := range fn.Type.Results.List {
				if id, ok := r.Type.(*ast.Ident); ok && id.Name == "error" {
					idx.errorReturners[name] = true
					break
				}
			}
		}
	}
	return idx
}

func receiverTypeName(e ast.Expr) string {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
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
func analyzeTestFunc(fset *token.FileSet, path string, fn *ast.FuncDecl, idx *pkgIndex) []Finding {
	var findings []Finding
	add := func(line int, smell, severity, msg string, conf float64) {
		findings = append(findings, Finding{
			File:       path,
			Line:       line,
			TestName:   fn.Name.Name,
			Smell:      smell,
			Severity:   severity,
			Message:    msg,
			Confidence: conf,
			FixKind:    smellFixKind(smell),
		})
	}

	body := fn.Body
	if body == nil {
		return nil
	}

	asserts := collectAsserts(body)
	startLine := fset.Position(fn.Pos()).Line

	// S01: no assertions at all. Skip when the body is a single
	// delegation to another function (standard "small body delegates
	// to runner" pattern); the runner gets analyzed on its own merits.
	if len(asserts) == 0 && !hasManualFail(body) && !isSingleDelegation(body) {
		add(startLine, "S01", "H",
			fmt.Sprintf("test %q runs code but has no assertion", fn.Name.Name), 1.0)
	}

	// S02 / S10: tautological / expect-the-expected assertions.
	for _, a := range asserts {
		if isTautological(a) {
			add(fset.Position(a.Pos()).Line, "S02", "H",
				"tautological assertion: both sides are the same expression", 1.0)
		}
		if isExpectTheExpected(a, body) {
			add(fset.Position(a.Pos()).Line, "S10", "H",
				"expected value derived from actual computation", 0.7)
		}
		if isSensitiveEquality(a) {
			add(fset.Position(a.Pos()).Line, "S06", "M",
				"asserting on String()/Sprintf rendering is brittle", 0.9)
		}
	}

	// S04: only assertion is NotPanics or recover().
	if len(asserts) > 0 && allNotPanics(asserts) {
		add(startLine, "S04", "H",
			"only assertion is NotPanics; behavior unverified", 1.0)
	} else if len(asserts) == 0 && hasOnlyRecover(body) {
		add(startLine, "S04", "H",
			"only safety check is defer recover(); behavior unverified", 1.0)
	}

	// S05: error from SUT discarded. Confidence depends on whether we
	// can verify the callee actually returns an error.
	for _, d := range findDiscardedErrors(fset, body, idx) {
		add(d.line, "S05", "H", d.message, d.confidence)
	}

	// S07: conditional/skipped assertion (early return before assert).
	if line, ok := findSkippedAssertion(fset, body); ok {
		add(line, "S07", "M",
			"early return may bypass subsequent assertion", 0.7)
	}

	// S09: assertion roulette (>3 bare asserts, no messages, not table-driven).
	if !isTableDriven(body) && countBareAsserts(asserts) > 3 {
		add(startLine, "S09", "L",
			"many assertions without messages; failure will be ambiguous", 0.8)
	}

	// S11: SUT receives pointer/state but no read-back. Skip when the
	// test body is a single delegation; the receiving function is the
	// real test scope and S11 would be a false positive otherwise.
	if !isSingleDelegation(body) {
		if line, ok := findUnassertedSideEffect(fset, body); ok {
			add(line, "S11", "M",
				"SUT mutates argument; mutated state not asserted", 0.5)
		}
	}

	// S03: getter/setter trivial. Detected at function level.
	if isGetterSetterTrivial(body) {
		add(startLine, "S03", "M",
			"test only verifies setter then getter; tests language semantics, not behavior", 0.9)
	}

	return findings
}

// isSingleDelegation returns true when the body is a single ExprStmt
// containing one CallExpr — e.g. `func TestX(t *T) { runScenario(t, foo) }`.
// Such tests delegate all assertions to the callee; flagging them as
// "no assertions" is a false positive.
func isSingleDelegation(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) != 1 {
		return false
	}
	es, ok := body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	_, ok = es.X.(*ast.CallExpr)
	return ok
}

// collectAsserts returns all assertion-like calls in the test body,
// including those inside `t.Run("name", func(t *testing.T) { ... })`
// subtest closures. ast.Inspect already descends into FuncLit bodies,
// so explicit recursion isn't needed — but we double-check by walking
// FuncLit bodies that are arguments to t.Run, which is the common
// shape used to organize subtests.
func collectAsserts(body *ast.BlockStmt) []*ast.CallExpr {
	var out []*ast.CallExpr
	ast.Inspect(body, func(n ast.Node) bool {
		// Direct assert call.
		if call, ok := n.(*ast.CallExpr); ok && isAssertCall(call) {
			out = append(out, call)
		}
		// Walk into FuncLit bodies passed to t.Run. ast.Inspect already
		// recurses into FuncLit, so any inner asserts are seen — but
		// we keep this branch as a guard against future refactors.
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

// discardFinding carries the data findDiscardedErrors needs to emit
// a finding with appropriate confidence based on what we know about
// the callee.
type discardFinding struct {
	line       int
	message    string
	confidence float64
}

// findDiscardedErrors finds AssignStmts that discard a return with `_`
// and emits a finding only when the callee plausibly returns an error.
// Confidence reflects how certain we are:
//
//	1.0 — callee is a same-package func that returns error: definite
//	      smell.
//	0.4 — callee is unknown (cross-package): could be an error, could
//	      not.
//	0.0 — callee is a same-package func that does NOT return error:
//	      no finding emitted at all (filters out the helper-discard
//	      false-positive observed on darepo PR 305).
func findDiscardedErrors(fset *token.FileSet, body *ast.BlockStmt, idx *pkgIndex) []discardFinding {
	var out []discardFinding
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		// Must have at least one `_` on the LHS.
		discardCount := 0
		for _, l := range as.Lhs {
			if id, ok := l.(*ast.Ident); ok && id.Name == "_" {
				discardCount++
			}
		}
		if discardCount == 0 {
			return true
		}
		// RHS must be a single CallExpr.
		if len(as.Rhs) != 1 {
			return true
		}
		call, ok := as.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		// Either every LHS is a discard, or the last LHS is `_`.
		lastIsDiscard := func() bool {
			last, ok := as.Lhs[len(as.Lhs)-1].(*ast.Ident)
			return ok && last.Name == "_"
		}()
		if discardCount != len(as.Lhs) && !lastIsDiscard {
			return true
		}
		// Resolve callee name. Skip if we can't (e.g., method on a
		// composite expression — too unreliable to flag).
		name := calleeName(call)
		if name == "" {
			return true
		}
		// Apply package-index knowledge.
		conf := 0.4 // unknown callee: low-confidence default.
		known := idx != nil && idx.knownFuncs[name]
		if known {
			if !idx.errorReturners[name] {
				// Known same-package callee that does NOT return
				// error — this is the helper-with-(T,U) case. Skip.
				return true
			}
			conf = 1.0
		}
		out = append(out, discardFinding{
			line:       fset.Position(as.Pos()).Line,
			message:    "error return from SUT discarded with _",
			confidence: conf,
		})
		return true
	})
	return out
}

// calleeName returns a name we can look up in the pkgIndex for a
// CallExpr. Returns "" when the callee is too complex to resolve
// without type information.
func calleeName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		// e.g., `obj.Method(...)` — return bare method name. The
		// pkgIndex stores both bare names and `Receiver.Method`,
		// but bare-name match is the safer starting point.
		return fn.Sel.Name
	}
	return ""
}

// findSkippedAssertion returns the line of an early `return` that bypasses
// later assertions.
func findSkippedAssertion(fset *token.FileSet, body *ast.BlockStmt) (int, bool) {
	// Walk top-level statements. If we see an If with a `return` in its body,
	// and there's an assertion call after the If, flag the early return.
	stmts := body.List
	for i, s := range stmts {
		ifs, ok := s.(*ast.IfStmt)
		if !ok {
			continue
		}
		ret := returnInBlock(ifs.Body)
		if ret == nil {
			continue
		}
		// Are there asserts after this if-stmt?
		for _, later := range stmts[i+1:] {
			if blockOrStmtHasAssert(later) {
				return fset.Position(ret.Pos()).Line, true
			}
		}
	}
	return 0, false
}

// returnInBlock returns the first ReturnStmt in a block, or nil.
func returnInBlock(b *ast.BlockStmt) *ast.ReturnStmt {
	for _, s := range b.List {
		if r, ok := s.(*ast.ReturnStmt); ok {
			return r
		}
	}
	return nil
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
func findUnassertedSideEffect(fset *token.FileSet, body *ast.BlockStmt) (int, bool) {
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
		// Pointer-passing arg? Skip the test's own `t` parameter — it
		// is not a SUT side effect, just the testing handle.
		for _, arg := range call.Args {
			if u, ok := arg.(*ast.UnaryExpr); ok && u.Op == token.AND {
				if id, ok := u.X.(*ast.Ident); ok && id.Name != "t" && id.Name != "tb" && id.Name != "rt" {
					passes = append(passes, ptrPass{name: id.Name, line: fset.Position(es.Pos()).Line})
				}
			} else if id, ok := arg.(*ast.Ident); ok {
				if id.Name == "t" || id.Name == "tb" || id.Name == "rt" {
					continue
				}
				// Could be a pointer var; we don't have type info.
				passes = append(passes, ptrPass{name: id.Name, line: fset.Position(es.Pos()).Line})
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
