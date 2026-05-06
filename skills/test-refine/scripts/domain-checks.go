// domain-checks scans a Go package for distributed-systems / Bitcoin /
// networking-specific test gaps:
//
//	D-CONCURRENCY-MISSING : SUT uses concurrency primitives but no test
//	                        exercises concurrent calls.
//	D-ERR-PATH-MISSING    : SUT returns error; no test asserts a specific
//	                        error case.
//	D-CTX-CANCEL-MISSING  : SUT takes context.Context; no cancellation test.
//	D-CTX-TIMEOUT-MISSING : Same, no timeout test.
//	D-PBT-CANDIDATE       : SUT pair admits a roundtrip / oracle property.
//	D-DETERMINISM-CLOCK   : Test reads time.Now() directly.
//	D-DETERMINISM-RAND    : Test uses unseeded rand.
//	D-DETERMINISM-ENV     : Test reads os.Getenv.
//
// Usage:
//
//	go run domain-checks.go --pkg ./internal/wallet
//
// Output: JSON-lines findings on stdout (same shape as detect-smells).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type Finding struct {
	File             string `json:"file"`
	Line             int    `json:"line"`
	TestName         string `json:"test_name,omitempty"`
	Smell            string `json:"smell"`
	Severity         string `json:"severity"`
	Message          string `json:"message"`
	FunctionUnderTest string `json:"function_under_test,omitempty"`
	Suggestion       string `json:"suggestion,omitempty"`
}

func main() {
	pkg := flag.String("pkg", ".", "package directory to analyze")
	flag.Parse()

	prodFiles, testFiles, err := splitGoFiles(*pkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fset := token.NewFileSet()
	prod := parseFiles(fset, prodFiles)
	tests := parseFiles(fset, testFiles)

	enc := json.NewEncoder(os.Stdout)
	emit := func(f Finding) { _ = enc.Encode(f) }

	// --- Determinism checks (in test files only) ---
	for path, file := range tests {
		walkDeterminism(fset, path, file, emit)
	}

	// --- Concurrency / failure-mode / PBT checks (cross prod x tests) ---
	prodFuncs := collectFuncs(prod)
	testFuncs := collectFuncs(tests)

	for fnKey, fn := range prodFuncs {
		// Function visibility: only public functions are easily targets.
		if !isExported(fn.fn.Name.Name) {
			continue
		}
		// Find tests that mention this function.
		mentionsFn := func(tf funcInfo) bool {
			return funcMentionsName(tf.fn, fn.fn.Name.Name)
		}
		var related []funcInfo
		for _, tf := range testFuncs {
			if mentionsFn(tf) {
				related = append(related, tf)
			}
		}

		// Concurrency: SUT uses goroutines/channels/sync? Tests must too.
		if usesConcurrency(fn) && !anyTestUsesConcurrency(related) {
			emit(Finding{
				File:              fn.path,
				Line:              fset.Position(fn.fn.Pos()).Line,
				Smell:             "D-CONCURRENCY-MISSING",
				Severity:          "M",
				Message:           "SUT uses concurrency primitives; no test exercises concurrent calls",
				FunctionUnderTest: fnKey,
				Suggestion:        "add a test that calls the SUT from multiple goroutines and runs with -race",
			})
		}

		// Error path missing.
		if returnsError(fn.fn) && !anyTestAssertsErrorIs(related) {
			emit(Finding{
				File:              fn.path,
				Line:              fset.Position(fn.fn.Pos()).Line,
				Smell:             "D-ERR-PATH-MISSING",
				Severity:          "M",
				Message:           "SUT returns error; no test asserts on a specific error case",
				FunctionUnderTest: fnKey,
				Suggestion:        "add tests using require.ErrorIs / require.ErrorAs against named sentinels",
			})
		}

		// Context cancellation / timeout.
		if takesContext(fn.fn) {
			if !anyTestUsesContextCancel(related) {
				emit(Finding{
					File:              fn.path,
					Line:              fset.Position(fn.fn.Pos()).Line,
					Smell:             "D-CTX-CANCEL-MISSING",
					Severity:          "M",
					Message:           "SUT takes context.Context; no test exercises cancellation",
					FunctionUnderTest: fnKey,
					Suggestion:        "add a test using context.WithCancel and asserting context.Canceled",
				})
			}
			if !anyTestUsesContextTimeout(related) {
				emit(Finding{
					File:              fn.path,
					Line:              fset.Position(fn.fn.Pos()).Line,
					Smell:             "D-CTX-TIMEOUT-MISSING",
					Severity:          "L",
					Message:           "SUT takes context.Context; no test exercises a timeout",
					FunctionUnderTest: fnKey,
					Suggestion:        "add a test using context.WithTimeout asserting context.DeadlineExceeded",
				})
			}
		}
	}

	// --- PBT candidate detection: look for Marshal/Unmarshal etc. pairs ---
	for fnKey, fn := range prodFuncs {
		base, kind := pairKind(fn.fn.Name.Name)
		if kind == "" {
			continue
		}
		other := pairOther(fnKey, base, kind, prodFuncs)
		if other != "" {
			emit(Finding{
				File:              fn.path,
				Line:              fset.Position(fn.fn.Pos()).Line,
				Smell:             "D-PBT-CANDIDATE",
				Severity:          "M",
				Message:           fmt.Sprintf("%s/%s pair admits a roundtrip property", fn.fn.Name.Name, baseName(other)),
				FunctionUnderTest: fnKey,
				Suggestion:        "use rapid to assert: " + roundtripFor(kind, fn.fn.Name.Name, baseName(other)),
			})
		}
	}
}

// --- File handling ---

func splitGoFiles(dir string) (prod, tests []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if strings.HasSuffix(e.Name(), "_test.go") {
			tests = append(tests, path)
		} else {
			prod = append(prod, path)
		}
	}
	return prod, tests, nil
}

func parseFiles(fset *token.FileSet, paths []string) map[string]*ast.File {
	m := map[string]*ast.File{}
	for _, p := range paths {
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", p, err)
			continue
		}
		m[p] = f
	}
	return m
}

type funcInfo struct {
	path string
	fn   *ast.FuncDecl
}

func collectFuncs(files map[string]*ast.File) map[string]funcInfo {
	out := map[string]funcInfo{}
	for path, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			key := fn.Name.Name
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				if t := receiverTypeName(fn.Recv.List[0].Type); t != "" {
					key = t + "." + fn.Name.Name
				}
			}
			out[key] = funcInfo{path: path, fn: fn}
		}
	}
	return out
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

// --- Predicates over functions ---

func isExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}

func returnsError(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, r := range fn.Type.Results.List {
		if id, ok := r.Type.(*ast.Ident); ok && id.Name == "error" {
			return true
		}
	}
	return false
}

func takesContext(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}
	for _, p := range fn.Type.Params.List {
		if sel, ok := p.Type.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "context" && sel.Sel.Name == "Context" {
				return true
			}
		}
	}
	return false
}

func usesConcurrency(fi funcInfo) bool {
	uses := false
	ast.Inspect(fi.fn.Body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.GoStmt:
			uses = true
			return false
		case *ast.SendStmt:
			uses = true
			return false
		case *ast.SelectStmt:
			uses = true
			return false
		case *ast.UnaryExpr:
			if v.Op == token.ARROW {
				uses = true
				return false
			}
		case *ast.SelectorExpr:
			if id, ok := v.X.(*ast.Ident); ok {
				switch id.Name {
				case "sync", "atomic":
					uses = true
					return false
				}
			}
		case *ast.ChanType:
			uses = true
			return false
		}
		return true
	})
	return uses
}

func funcMentionsName(fn *ast.FuncDecl, name string) bool {
	if fn.Body == nil {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == name {
			found = true
			return false
		}
		if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

// --- Predicates over related tests ---

func anyTestUsesConcurrency(tests []funcInfo) bool {
	for _, t := range tests {
		if usesConcurrency(t) {
			return true
		}
	}
	return false
}

func anyTestAssertsErrorIs(tests []funcInfo) bool {
	for _, t := range tests {
		if funcCallsName(t.fn, "ErrorIs") || funcCallsName(t.fn, "ErrorAs") {
			return true
		}
	}
	return false
}

func anyTestUsesContextCancel(tests []funcInfo) bool {
	for _, t := range tests {
		if funcCallsSelector(t.fn, "context", "WithCancel") {
			return true
		}
	}
	return false
}

func anyTestUsesContextTimeout(tests []funcInfo) bool {
	for _, t := range tests {
		if funcCallsSelector(t.fn, "context", "WithTimeout") ||
			funcCallsSelector(t.fn, "context", "WithDeadline") {
			return true
		}
	}
	return false
}

func funcCallsName(fn *ast.FuncDecl, name string) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == name {
			found = true
			return false
		}
		if id, ok := c.Fun.(*ast.Ident); ok && id.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

func funcCallsSelector(fn *ast.FuncDecl, pkg, name string) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if id.Name == pkg && sel.Sel.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

// --- Determinism checks in test files ---

func walkDeterminism(fset *token.FileSet, path string, file *ast.File, emit func(Finding)) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		// Only apply to test funcs.
		if !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
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
			id, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			pos := fset.Position(c.Pos()).Line
			switch id.Name {
			case "time":
				if sel.Sel.Name == "Now" {
					emit(Finding{
						File: path, Line: pos, TestName: fn.Name.Name,
						Smell: "D-DETERMINISM-CLOCK", Severity: "M",
						Message:    "test reads time.Now() directly; non-deterministic",
						Suggestion: "inject a clock interface; use clock.NewMock() in tests",
					})
				}
			case "rand":
				if sel.Sel.Name == "Int" || sel.Sel.Name == "Intn" ||
					sel.Sel.Name == "Int63" || sel.Sel.Name == "Int63n" ||
					sel.Sel.Name == "Float64" || sel.Sel.Name == "Read" {
					emit(Finding{
						File: path, Line: pos, TestName: fn.Name.Name,
						Smell: "D-DETERMINISM-RAND", Severity: "M",
						Message:    "test uses package-level math/rand; not seeded for reproducibility",
						Suggestion: "use rand.New(rand.NewSource(seed)); print seed on failure for replay",
					})
				}
			case "os":
				if sel.Sel.Name == "Getenv" {
					emit(Finding{
						File: path, Line: pos, TestName: fn.Name.Name,
						Smell: "D-DETERMINISM-ENV", Severity: "L",
						Message:    "test reads os.Getenv; depends on shell environment",
						Suggestion: "use t.Setenv to control the env, or pass the value through a struct",
					})
				}
			}
			return true
		})
	}
}

// --- PBT pair detection ---

// pairKind returns (base, kind) if name fits one of the known patterns.
// kind is one of "marshal", "encode", "format", "compose".
func pairKind(name string) (string, string) {
	switch {
	case strings.HasPrefix(name, "Marshal"):
		return strings.TrimPrefix(name, "Marshal"), "marshal"
	case strings.HasPrefix(name, "Unmarshal"):
		return strings.TrimPrefix(name, "Unmarshal"), "unmarshal"
	case strings.HasPrefix(name, "Encode"):
		return strings.TrimPrefix(name, "Encode"), "encode"
	case strings.HasPrefix(name, "Decode"):
		return strings.TrimPrefix(name, "Decode"), "decode"
	case strings.HasPrefix(name, "Parse"):
		return strings.TrimPrefix(name, "Parse"), "parse"
	case strings.HasPrefix(name, "Format"):
		return strings.TrimPrefix(name, "Format"), "format"
	}
	return "", ""
}

// pairOther returns the matching peer's key, if any.
func pairOther(self, base, kind string, all map[string]funcInfo) string {
	var candidates []string
	switch kind {
	case "marshal":
		candidates = []string{"Unmarshal" + base}
	case "unmarshal":
		candidates = []string{"Marshal" + base}
	case "encode":
		candidates = []string{"Decode" + base}
	case "decode":
		candidates = []string{"Encode" + base}
	case "parse":
		candidates = []string{"Format" + base, "String" + base}
	case "format":
		candidates = []string{"Parse" + base}
	}
	// Strip method receiver prefix when looking up.
	for _, c := range candidates {
		for k := range all {
			if k == c || strings.HasSuffix(k, "."+c) {
				return k
			}
		}
	}
	return ""
}

func baseName(key string) string {
	if i := strings.LastIndex(key, "."); i >= 0 {
		return key[i+1:]
	}
	return key
}

func roundtripFor(kind, a, b string) string {
	switch kind {
	case "marshal":
		return fmt.Sprintf("%s(%s(x)) == x", b, a)
	case "encode":
		return fmt.Sprintf("%s(%s(x)) == x", b, a)
	case "parse":
		return fmt.Sprintf("%s(%s(x)) == x", a, b)
	case "format":
		return fmt.Sprintf("%s(%s(x)) == x", b, a)
	}
	return fmt.Sprintf("%s(%s(x)) == x", b, a)
}
