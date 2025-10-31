package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// Mutation represents a single mutation to be applied.
type Mutation struct {
	ID          string `json:"id"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Type        string `json:"type"`
	Original    string `json:"original"`
	Mutated     string `json:"mutated"`
	Description string `json:"description"`
	Priority    string `json:"priority"` // high, medium, low
}

var (
	file     = flag.String("file", "", "Go source file to analyze")
	output   = flag.String("output", "", "Output file for mutations (default: stdout)")
	function = flag.String("function", "", "Only generate mutations for specific function")
	lines    = flag.String("lines", "", "Only generate mutations for line range (e.g., 100-200)")
)

func main() {
	flag.Parse()

	if *file == "" {
		fmt.Fprintln(os.Stderr, "Error: --file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the Go source file.
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, *file, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Generate mutations.
	generator := &MutationGenerator{
		fset:         fset,
		file:         *file,
		mutations:    []Mutation{},
		mutationID:   0,
		targetFunc:   *function,
		targetLines:  parseLineRange(*lines),
	}

	ast.Inspect(node, generator.visit)

	// Output mutations as JSON.
	outputMutations(generator.mutations, *output)
}

// MutationGenerator walks the AST and generates mutations.
type MutationGenerator struct {
	fset         *token.FileSet
	file         string
	mutations    []Mutation
	mutationID   int
	targetFunc   string
	targetLines  [2]int // [start, end], 0 means no filter
	currentFunc  string
}

func (g *MutationGenerator) visit(node ast.Node) bool {
	if node == nil {
		return false
	}

	// Track current function name.
	if fn, ok := node.(*ast.FuncDecl); ok {
		g.currentFunc = fn.Name.Name
		// If targeting specific function, skip others.
		if g.targetFunc != "" && g.currentFunc != g.targetFunc {
			return false
		}
	}

	pos := g.fset.Position(node.Pos())

	// If targeting specific line range, skip nodes outside range.
	if g.targetLines[0] > 0 {
		if pos.Line < g.targetLines[0] || pos.Line > g.targetLines[1] {
			return true // Continue walking but don't mutate this node.
		}
	}

	// Generate mutations based on node type.
	switch n := node.(type) {
	case *ast.BinaryExpr:
		g.mutateBinaryExpr(n)
	case *ast.UnaryExpr:
		g.mutateUnaryExpr(n)
	case *ast.BasicLit:
		g.mutateBasicLit(n)
	case *ast.ReturnStmt:
		g.mutateReturnStmt(n)
	case *ast.IfStmt:
		g.mutateIfStmt(n)
	case *ast.AssignStmt:
		g.mutateAssignStmt(n)
	case *ast.IncDecStmt:
		g.mutateIncDecStmt(n)
	}

	return true
}

// mutateBinaryExpr generates mutations for binary expressions.
func (g *MutationGenerator) mutateBinaryExpr(expr *ast.BinaryExpr) {
	pos := g.fset.Position(expr.Pos())
	original := g.tokenToString(expr.Op)

	// Arithmetic operator mutations.
	if isArithmetic(expr.Op) {
		for _, mutOp := range arithmeticMutations(expr.Op) {
			g.addMutation(Mutation{
				File:        g.file,
				Line:        pos.Line,
				Column:      pos.Column,
				Type:        "arithmetic_operator",
				Original:    original,
				Mutated:     g.tokenToString(mutOp),
				Description: fmt.Sprintf("Change %s to %s", original, g.tokenToString(mutOp)),
				Priority:    "high",
			})
		}
	}

	// Relational operator mutations.
	if isRelational(expr.Op) {
		for _, mutOp := range relationalMutations(expr.Op) {
			priority := "high" // Boundary conditions are critical.
			g.addMutation(Mutation{
				File:        g.file,
				Line:        pos.Line,
				Column:      pos.Column,
				Type:        "relational_operator",
				Original:    original,
				Mutated:     g.tokenToString(mutOp),
				Description: fmt.Sprintf("Change %s to %s (boundary)", original, g.tokenToString(mutOp)),
				Priority:    priority,
			})
		}
	}

	// Logical operator mutations.
	if isLogical(expr.Op) {
		for _, mutOp := range logicalMutations(expr.Op) {
			g.addMutation(Mutation{
				File:        g.file,
				Line:        pos.Line,
				Column:      pos.Column,
				Type:        "logical_operator",
				Original:    original,
				Mutated:     g.tokenToString(mutOp),
				Description: fmt.Sprintf("Change %s to %s", original, g.tokenToString(mutOp)),
				Priority:    "high", // Logical operators are security-critical.
			})
		}
	}
}

// mutateUnaryExpr generates mutations for unary expressions.
func (g *MutationGenerator) mutateUnaryExpr(expr *ast.UnaryExpr) {
	pos := g.fset.Position(expr.Pos())
	original := g.tokenToString(expr.Op)

	// Negation removal: !x → x
	if expr.Op == token.NOT {
		g.addMutation(Mutation{
			File:        g.file,
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "unary_operator",
			Original:    original,
			Mutated:     "(removed)",
			Description: "Remove negation operator",
			Priority:    "high",
		})
	}

	// Unary minus/plus mutations.
	if expr.Op == token.SUB {
		g.addMutation(Mutation{
			File:        g.file,
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "unary_operator",
			Original:    "-",
			Mutated:     "+",
			Description: "Change unary minus to plus",
			Priority:    "medium",
		})
	}
}

// mutateBasicLit generates mutations for basic literals.
func (g *MutationGenerator) mutateBasicLit(lit *ast.BasicLit) {
	pos := g.fset.Position(lit.Pos())

	switch lit.Kind {
	case token.INT:
		// Integer constant mutations.
		if lit.Value == "0" {
			g.addMutation(Mutation{
				File:        g.file,
				Line:        pos.Line,
				Column:      pos.Column,
				Type:        "constant",
				Original:    "0",
				Mutated:     "1",
				Description: "Change 0 to 1",
				Priority:    "high",
			})
		} else if lit.Value == "1" {
			g.addMutation(Mutation{
				File:        g.file,
				Line:        pos.Line,
				Column:      pos.Column,
				Type:        "constant",
				Original:    "1",
				Mutated:     "0",
				Description: "Change 1 to 0",
				Priority:    "high",
			})
		} else {
			// Generic numeric mutation (off-by-one).
			g.addMutation(Mutation{
				File:        g.file,
				Line:        pos.Line,
				Column:      pos.Column,
				Type:        "constant",
				Original:    lit.Value,
				Mutated:     fmt.Sprintf("(%s + 1)", lit.Value),
				Description: fmt.Sprintf("Off-by-one: %s → %s + 1", lit.Value, lit.Value),
				Priority:    "medium",
			})
		}

	case token.STRING:
		// String constant mutations.
		if lit.Value != `""` {
			g.addMutation(Mutation{
				File:        g.file,
				Line:        pos.Line,
				Column:      pos.Column,
				Type:        "constant",
				Original:    lit.Value,
				Mutated:     `""`,
				Description: "Change string to empty",
				Priority:    "medium",
			})
		}
	}
}

// mutateReturnStmt generates mutations for return statements.
func (g *MutationGenerator) mutateReturnStmt(stmt *ast.ReturnStmt) {
	pos := g.fset.Position(stmt.Pos())

	// Return value mutations.
	if len(stmt.Results) > 0 {
		for i, result := range stmt.Results {
			// Boolean return mutations.
			if ident, ok := result.(*ast.Ident); ok {
				if ident.Name == "true" {
					g.addMutation(Mutation{
						File:        g.file,
						Line:        pos.Line,
						Column:      pos.Column,
						Type:        "return_value",
						Original:    "true",
						Mutated:     "false",
						Description: fmt.Sprintf("Change return value %d: true → false", i),
						Priority:    "high",
					})
				} else if ident.Name == "false" {
					g.addMutation(Mutation{
						File:        g.file,
						Line:        pos.Line,
						Column:      pos.Column,
						Type:        "return_value",
						Original:    "false",
						Mutated:     "true",
						Description: fmt.Sprintf("Change return value %d: false → true", i),
						Priority:    "high",
					})
				} else if ident.Name == "nil" {
					g.addMutation(Mutation{
						File:        g.file,
						Line:        pos.Line,
						Column:      pos.Column,
						Type:        "return_value",
						Original:    "nil",
						Mutated:     "non-nil",
						Description: fmt.Sprintf("Change return value %d: nil → non-nil", i),
						Priority:    "high",
					})
				}
			}
		}
	}

	// Statement removal: remove entire return.
	g.addMutation(Mutation{
		File:        g.file,
		Line:        pos.Line,
		Column:      pos.Column,
		Type:        "statement_removal",
		Original:    "return statement",
		Mutated:     "(removed)",
		Description: "Remove return statement",
		Priority:    "medium",
	})
}

// mutateIfStmt generates mutations for if statements.
func (g *MutationGenerator) mutateIfStmt(stmt *ast.IfStmt) {
	pos := g.fset.Position(stmt.Pos())

	// Condition negation.
	g.addMutation(Mutation{
		File:        g.file,
		Line:        pos.Line,
		Column:      pos.Column,
		Type:        "condition_negation",
		Original:    "if condition",
		Mutated:     "if !condition",
		Description: "Negate if condition",
		Priority:    "high",
	})

	// Else branch removal (if exists).
	if stmt.Else != nil {
		g.addMutation(Mutation{
			File:        g.file,
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "else_removal",
			Original:    "else branch",
			Mutated:     "(removed)",
			Description: "Remove else branch",
			Priority:    "medium",
		})
	}
}

// mutateAssignStmt generates mutations for assignment statements.
func (g *MutationGenerator) mutateAssignStmt(stmt *ast.AssignStmt) {
	pos := g.fset.Position(stmt.Pos())

	// Compound assignment mutations.
	if stmt.Tok != token.ASSIGN && stmt.Tok != token.DEFINE {
		original := g.tokenToString(stmt.Tok)

		// += to -=, etc.
		if stmt.Tok == token.ADD_ASSIGN {
			g.addMutation(Mutation{
				File:        g.file,
				Line:        pos.Line,
				Column:      pos.Column,
				Type:        "assignment_operator",
				Original:    original,
				Mutated:     "-=",
				Description: "Change += to -=",
				Priority:    "high",
			})
		} else if stmt.Tok == token.SUB_ASSIGN {
			g.addMutation(Mutation{
				File:        g.file,
				Line:        pos.Line,
				Column:      pos.Column,
				Type:        "assignment_operator",
				Original:    original,
				Mutated:     "+=",
				Description: "Change -= to +=",
				Priority:    "high",
			})
		}
	}
}

// mutateIncDecStmt generates mutations for increment/decrement statements.
func (g *MutationGenerator) mutateIncDecStmt(stmt *ast.IncDecStmt) {
	pos := g.fset.Position(stmt.Pos())

	if stmt.Tok == token.INC {
		g.addMutation(Mutation{
			File:        g.file,
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "inc_dec",
			Original:    "++",
			Mutated:     "--",
			Description: "Change ++ to --",
			Priority:    "high",
		})
	} else if stmt.Tok == token.DEC {
		g.addMutation(Mutation{
			File:        g.file,
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "inc_dec",
			Original:    "--",
			Mutated:     "++",
			Description: "Change -- to ++",
			Priority:    "high",
		})
	}
}

// Helper functions.

func (g *MutationGenerator) addMutation(m Mutation) {
	m.ID = fmt.Sprintf("M%d", g.mutationID)
	g.mutationID++
	g.mutations = append(g.mutations, m)
}

func (g *MutationGenerator) tokenToString(tok token.Token) string {
	return tok.String()
}

func isArithmetic(op token.Token) bool {
	return op == token.ADD || op == token.SUB || op == token.MUL || op == token.QUO || op == token.REM
}

func isRelational(op token.Token) bool {
	return op == token.LSS || op == token.GTR || op == token.LEQ || op == token.GEQ || op == token.EQL || op == token.NEQ
}

func isLogical(op token.Token) bool {
	return op == token.LAND || op == token.LOR
}

func arithmeticMutations(op token.Token) []token.Token {
	switch op {
	case token.ADD:
		return []token.Token{token.SUB, token.MUL, token.QUO}
	case token.SUB:
		return []token.Token{token.ADD, token.MUL, token.QUO}
	case token.MUL:
		return []token.Token{token.ADD, token.SUB, token.QUO}
	case token.QUO:
		return []token.Token{token.ADD, token.SUB, token.MUL}
	case token.REM:
		return []token.Token{token.QUO, token.MUL}
	}
	return nil
}

func relationalMutations(op token.Token) []token.Token {
	switch op {
	case token.LSS: // <
		return []token.Token{token.LEQ, token.GTR, token.GEQ, token.EQL, token.NEQ}
	case token.GTR: // >
		return []token.Token{token.GEQ, token.LSS, token.LEQ, token.EQL, token.NEQ}
	case token.LEQ: // <=
		return []token.Token{token.LSS, token.GEQ, token.GTR, token.EQL, token.NEQ}
	case token.GEQ: // >=
		return []token.Token{token.GTR, token.LEQ, token.LSS, token.EQL, token.NEQ}
	case token.EQL: // ==
		return []token.Token{token.NEQ, token.LSS, token.GTR}
	case token.NEQ: // !=
		return []token.Token{token.EQL}
	}
	return nil
}

func logicalMutations(op token.Token) []token.Token {
	switch op {
	case token.LAND: // &&
		return []token.Token{token.LOR}
	case token.LOR: // ||
		return []token.Token{token.LAND}
	}
	return nil
}

func parseLineRange(rangeStr string) [2]int {
	if rangeStr == "" {
		return [2]int{0, 0}
	}

	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "Invalid line range: %s (expected format: 100-200)\n", rangeStr)
		os.Exit(1)
	}

	var start, end int
	fmt.Sscanf(parts[0], "%d", &start)
	fmt.Sscanf(parts[1], "%d", &end)
	return [2]int{start, end}
}

func outputMutations(mutations []Mutation, outputFile string) {
	data, err := json.MarshalIndent(mutations, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling mutations: %v\n", err)
		os.Exit(1)
	}

	if outputFile == "" {
		fmt.Println(string(data))
	} else {
		err := os.WriteFile(outputFile, data, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Generated %d mutations → %s\n", len(mutations), outputFile)
	}
}
