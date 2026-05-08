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
	File              string  `json:"file"`
	Line              int     `json:"line"`
	TestName          string  `json:"test_name"`
	Smell             string  `json:"smell"`
	Severity          string  `json:"severity"`
	Message           string  `json:"message"`
	Confidence        float64 `json:"confidence,omitempty"`
	FixKind           string  `json:"fix_kind,omitempty"`
	Context           string  `json:"context,omitempty"`
	Suggestion        string  `json:"suggestion,omitempty"`
	FunctionUnderTest string  `json:"function_under_test,omitempty"`
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
// Sourced from testify/{require,assert} v1 — kept in sync because a
// missing entry causes S01 to flag tests that have assertions.
var assertCalls = map[string]bool{
	// Equality.
	"Equal": true, "NotEqual": true,
	"EqualValues": true, "NotEqualValues": true,
	"Exactly": true,
	// Booleans.
	"True": true, "False": true,
	// Nilness.
	"Nil": true, "NotNil": true,
	"Zero": true, "NotZero": true,
	// Emptiness.
	"Empty": true, "NotEmpty": true,
	// Errors. ErrorContains / ErrorContainsf / EqualError / EqualErrorf
	// were missing in earlier revisions and caused S01 false positives
	// on tests that asserted exclusively via require.ErrorContains.
	"Error": true, "NoError": true,
	"Errorf": true, "NoErrorf": true,
	"ErrorIs": true, "ErrorAs": true, "NotErrorIs": true,
	"ErrorContains": true, "ErrorContainsf": true,
	"EqualError": true, "EqualErrorf": true,
	"PanicsWithError": true, "PanicsWithValue": true,
	// Containment / collections.
	"Contains": true, "NotContains": true,
	"Subset": true, "NotSubset": true,
	"ElementsMatch": true,
	"Len": true,
	// Ordering / numeric.
	"Greater": true, "Less": true,
	"GreaterOrEqual": true, "LessOrEqual": true,
	"Positive": true, "Negative": true,
	"InDelta": true, "InEpsilon": true,
	"InDeltaSlice": true, "InEpsilonSlice": true,
	"InDeltaMapValues": true,
	// Panics.
	"Panics": true, "NotPanics": true,
	// Identity / reflection.
	"Same": true, "NotSame": true,
	"IsType": true, "IsNotType": true,
	"Implements": true,
	// JSON / YAML / regex.
	"JSONEq": true, "YAMLEq": true,
	"Regexp": true, "NotRegexp": true,
	// Async.
	"Eventually": true, "EventuallyWithT": true, "Never": true,
	// FS.
	"FileExists": true, "DirExists": true,
	"NoFileExists": true, "NoDirExists": true,
	// Time / HTTP — included for completeness; rarely used in unit tests.
	"WithinDuration": true,
	"HTTPSuccess":    true, "HTTPRedirect": true,
	"HTTPError": true, "HTTPStatusCode": true,
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

	// First pass: parse every .go input (test + production) into the
	// shared index. Production sources are essential for SUT-name
	// resolution: `TestActorStart` -> `Actor.Start` only works when
	// `Actor.Start` is in knownFuncs. Test files are kept in `testFiles`
	// for the analysis pass; production files contribute to the index
	// only.
	var testFiles, indexFiles []parsedFile
	for _, path := range os.Args[1:] {
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, err)
			continue
		}
		pf := parsedFile{path: path, fset: fset, file: file}
		indexFiles = append(indexFiles, pf)
		if strings.HasSuffix(path, "_test.go") {
			testFiles = append(testFiles, pf)
		}
	}

	idx := buildPkgIndex(indexFiles)

	// Second pass: analyze each test file. Capture per-test style so
	// we can run the package-wide drift detector after.
	enc := json.NewEncoder(os.Stdout)
	var styles []testStyle

	for _, p := range testFiles {
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
	sut := deriveSUT(fn, idx)
	add := func(line int, smell, severity, msg string, conf float64) {
		findings = append(findings, Finding{
			File:              path,
			Line:              line,
			TestName:          fn.Name.Name,
			Smell:             smell,
			Severity:          severity,
			Message:           msg,
			Confidence:        conf,
			FixKind:           smellFixKind(smell),
			FunctionUnderTest: sut,
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

	// S09: assertion roulette. The original heuristic — bare-assert count
	// > 3 — produced massive volume on real codebases (24 of the top 30
	// findings in one user run), because Go's require.X already prints
	// file:line and the values. The risk only materialises when several
	// asserts share the same shape AND would collapse into the same
	// failure message. We now require:
	//
	//   1. The test isn't table-driven.
	//   2. There are at least two bare asserts whose (call-name, RHS
	//      canonical text) match — i.e. a real ambiguity, not "five
	//      different require.Equal calls on different fields".
	//   3. There are at least 5 bare asserts in total.
	//
	// Confidence is lowered to 0.5 to reflect that even matched-shape
	// asserts often have distinguishing line numbers in the failure.
	// Even with deduplication the heuristic stays low-value: `go test`
	// prints file:line + values for any failed require.X, so a
	// distinguishable line is rarely truly ambiguous. We require both
	// a high overall bare-assert count *and* a tight cluster of
	// shape-duplicates before firing, plus a confidence well below the
	// auto-trust threshold.
	if !isTableDriven(body) {
		bareCount, dupGroup := analyzeBareAsserts(asserts)
		if bareCount >= 8 && dupGroup >= 4 {
			add(startLine, "S09", "L",
				"many shape-duplicated assertions without messages; failure may be ambiguous", 0.4)
		}
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

// deriveSUT picks a best-effort label for the function under test by
// matching the test name against the package-wide function index.
// `TestExtractCheckpointTxRejectsMalformed` becomes `ExtractCheckpointTx`
// when that function exists in the package; `TestActorStartFailClosed`
// becomes `Actor.Start` when `Actor.Start` is in the index.
//
// Returns "" when no plausible match exists — an empty label is more
// honest than a body-scan fallback, which (without type info) almost
// always surfaces harness setup or factory calls instead of the SUT.
func deriveSUT(fn *ast.FuncDecl, idx *pkgIndex) string {
	if fn == nil || fn.Name == nil {
		return ""
	}
	return matchSUTFromTestName(fn.Name.Name, idx)
}

// matchSUTFromTestName strips the Test prefix and tries:
//   - longest exact match against knownFuncs (unqualified function);
//   - any camel-case split that produces "Receiver.Method" present in
//     the index (longest receiver wins).
func matchSUTFromTestName(testName string, idx *pkgIndex) string {
	if idx == nil || !strings.HasPrefix(testName, "Test") {
		return ""
	}
	rest := testName[4:]
	if rest == "" {
		return ""
	}
	// Pass 1: longest unqualified prefix match.
	for end := len(rest); end > 0; end-- {
		candidate := rest[:end]
		// Skip single-letter matches — they're almost always
		// false hits like "T" → some local type T.
		if len(candidate) < 2 {
			continue
		}
		if idx.knownFuncs[candidate] {
			return candidate
		}
	}
	// Pass 2: receiver.method matches. Walk camel-case word
	// boundaries, prefer longest receiver that yields an index hit.
	parts := splitCamelCase(rest)
	for i := 1; i < len(parts); i++ {
		recv := strings.Join(parts[:i], "")
		// Try every method-suffix length so multi-word methods
		// (e.g. Start, StartFailClosed) are both attempted.
		for j := len(parts); j > i; j-- {
			method := strings.Join(parts[i:j], "")
			key := recv + "." + method
			if idx.knownFuncs[key] {
				return key
			}
		}
	}
	return ""
}

// splitCamelCase breaks an identifier on uppercase boundaries, treating
// runs of uppercase as a single segment when followed by another
// uppercase (so "HTTPServer" → ["HTTP", "Server"], not single-letter).
func splitCamelCase(s string) []string {
	var parts []string
	start := 0
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			// Boundary if previous char was lowercase, OR this is
			// the last char of an uppercase run followed by lower.
			prev := s[i-1]
			isPrevLower := prev >= 'a' && prev <= 'z'
			next := byte(0)
			if i+1 < len(s) {
				next = s[i+1]
			}
			isNextLower := next >= 'a' && next <= 'z'
			if isPrevLower || isNextLower {
				parts = append(parts, s[start:i])
				start = i
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
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

// helperPrefixes are common test-helper-name prefixes. A call like
// h.timeoutActor.assertRecurringTickScheduled(t, ...) is an assertion
// site even though we don't see require.X / assert.X. Without this, S01
// fires on tests that delegate all their checks to an extracted helper.
//
// Including both lowercase and uppercase variants so unexported helpers
// (`assertX`) and exported helpers (`AssertX`) both count.
var helperPrefixes = []string{
	"assert", "require", "verify", "expect", "check", "must",
	"Assert", "Require", "Verify", "Expect", "Check", "Must",
}

// isHelperAssertName returns true when name looks like a test-helper
// assertion call: a known prefix followed by an upper-case letter, an
// underscore, or end-of-string. Matches `assertX`, `verifyState`,
// `mustEqual`, `requireNonNil`. Does not match `expected`, `muster`,
// `verified`, `assertion`.
func isHelperAssertName(name string) bool {
	for _, p := range helperPrefixes {
		if !strings.HasPrefix(name, p) {
			continue
		}
		if len(name) == len(p) {
			return true
		}
		next := name[len(p)]
		if next == '_' || (next >= 'A' && next <= 'Z') {
			return true
		}
	}
	return false
}

// isAssertCall matches assert.X(...) / require.X(...) / t.Errorf, plus
// user-defined helper calls whose name follows the assertX / verifyX
// convention (regardless of receiver — `h.assertX(...)` counts).
func isAssertCall(call *ast.CallExpr) bool {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		name := fn.Sel.Name
		if assertCalls[name] || tErrorCalls[name] {
			return true
		}
		// Helper method on any receiver: h.assertX, foo.verifyY.
		if isHelperAssertName(name) {
			return true
		}
	case *ast.Ident:
		// Bare helper call: assertX(t, ...) defined in same file.
		if isHelperAssertName(fn.Name) {
			return true
		}
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

// canonicalStringReceivers names types whose .String() is the canonical
// comparison form, not a brittle "rendering". Equality on these is the
// idiomatic way to compare hashes / UUIDs / big ints / decimals — flagging
// them as S06 was a false positive.
//
// Matching is by **identifier suffix**: we look at the receiver expression
// of the .String() call. If it ends with one of these names (e.g.
// `outpoint.Hash` ends with `Hash`), the assertion is canonical. The
// suffix-match is an unavoidable approximation in the absence of type
// information; chosen names err on the side of "definitely canonical"
// so the heuristic stays conservative.
var canonicalStringReceiverSuffixes = []string{
	"Hash",     // chainhash.Hash, sha256, etc.
	"Hash32",   // 32-byte hash variants.
	"UUID",     // canonical hex form.
	"Int",      // big.Int.
	"Rat",      // big.Rat.
	"Float",    // big.Float.
	"Time",     // time.Time RFC3339.
	"Duration", // time.Duration.
	"PubKey",   // btcec / secp pubkeys serialise canonically.
	"Address",  // btcutil.Address derivatives.
	"OutPoint", // hash:idx canonical form.
	"TxID",     // 32-byte hex.
	"BatchID",  // generic batch identifier types.
	"NodeID",   // peer/node identifiers.
}

// isCanonicalStringReceiver returns true when the .String() call
// is invoked on an expression whose final selector / ident name has one
// of the canonical-receiver suffixes.
func isCanonicalStringReceiver(recv ast.Expr) bool {
	var name string
	switch r := recv.(type) {
	case *ast.Ident:
		name = r.Name
	case *ast.SelectorExpr:
		name = r.Sel.Name
	case *ast.CallExpr:
		// e.g. `getHash(x).String()` — peek at the call's function name.
		if sel, ok := r.Fun.(*ast.SelectorExpr); ok {
			name = sel.Sel.Name
		} else if id, ok := r.Fun.(*ast.Ident); ok {
			name = id.Name
		}
	default:
		return false
	}
	for _, suf := range canonicalStringReceiverSuffixes {
		if name == suf || strings.HasSuffix(name, suf) {
			return true
		}
	}
	return false
}

// isSensitiveEquality flags assert.Equal where one arg is fmt.Sprintf /
// fmt.Sprint / .String() — but skips canonical .String() types where
// hex/UUID equality is the idiomatic comparison.
func isSensitiveEquality(a *ast.CallExpr) bool {
	sel, ok := a.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Equal" {
		return false
	}
	args := skipTArg(a.Args)
	for _, arg := range args {
		c, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}
		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		// fmt.Sprint* — always brittle.
		if id, ok := s.X.(*ast.Ident); ok && id.Name == "fmt" &&
			(s.Sel.Name == "Sprint" || s.Sel.Name == "Sprintf" || s.Sel.Name == "Sprintln") {
			return true
		}
		// .String() — only brittle when the receiver isn't a type
		// whose .String() is its canonical comparison form.
		if s.Sel.Name == "String" && !isCanonicalStringReceiver(s.X) {
			return true
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
//	      no finding emitted at all (filters out helper-discard
//	      false-positives on tests with multi-return helpers).
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
	n, _ := analyzeBareAsserts(asserts)
	return n
}

// analyzeBareAsserts returns (totalBareAsserts, maxRepeatedShape).
// "Bare" = no trailing message argument. Two asserts share a shape when
// their call name and the canonical text of their RHS expression match
// — that's the case where a failure message can't tell them apart.
func analyzeBareAsserts(asserts []*ast.CallExpr) (total, maxRepeat int) {
	shapes := map[string]int{}
	for _, a := range asserts {
		sel, ok := a.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		args := skipTArg(a.Args)
		// Per-assertion bare arity (msg-less arg count).
		var bareArity int
		switch sel.Sel.Name {
		case "Equal", "EqualValues", "NotEqual":
			bareArity = 2
		case "True", "False", "Nil", "NotNil", "Empty", "NotEmpty",
			"Error", "NoError":
			bareArity = 1
		default:
			continue
		}
		if len(args) != bareArity {
			continue
		}
		total++
		// Build a canonical shape key. We use the last arg's text
		// (the "got" side in testify's want/got order) as the
		// distinguishing element. If two bare asserts have the same
		// call name AND same RHS text, a failure won't disambiguate.
		key := sel.Sel.Name + "::" + exprText(args[len(args)-1])
		shapes[key]++
		if shapes[key] > maxRepeat {
			maxRepeat = shapes[key]
		}
	}
	return total, maxRepeat
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
