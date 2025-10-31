package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// Mutation represents a single mutation descriptor.
type Mutation struct {
	ID          string `json:"id"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Type        string `json:"type"`
	Original    string `json:"original"`
	Mutated     string `json:"mutated"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

var (
	mutationFile = flag.String("mutation-file", "", "JSON file containing mutations")
	mutationID   = flag.String("mutation-id", "", "Specific mutation ID to apply (e.g., M0)")
	output       = flag.String("output", "", "Output file (default: stdout)")
	verify       = flag.Bool("verify", true, "Verify mutated code compiles")
)

func main() {
	flag.Parse()

	if *mutationFile == "" || *mutationID == "" {
		fmt.Fprintln(os.Stderr, "Error: --mutation-file and --mutation-id are required")
		flag.Usage()
		os.Exit(1)
	}

	// Load mutations from JSON file.
	mutations, err := loadMutations(*mutationFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading mutations: %v\n", err)
		os.Exit(1)
	}

	// Find the specific mutation.
	var mutation *Mutation
	for i := range mutations {
		if mutations[i].ID == *mutationID {
			mutation = &mutations[i]
			break
		}
	}

	if mutation == nil {
		fmt.Fprintf(os.Stderr, "Error: mutation %s not found\n", *mutationID)
		os.Exit(1)
	}

	// Apply the mutation.
	mutatedCode, err := applyMutation(mutation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error applying mutation: %v\n", err)
		os.Exit(1)
	}

	// Verify mutated code compiles (if requested).
	if *verify {
		if err := verifyCode(mutatedCode); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: mutated code has syntax errors: %v\n", err)
			// Continue anyway - some mutations may intentionally break syntax.
		}
	}

	// Output mutated code.
	if *output == "" {
		fmt.Print(mutatedCode)
	} else {
		err := os.WriteFile(*output, []byte(mutatedCode), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Applied mutation %s → %s\n", mutation.ID, *output)
	}
}

func loadMutations(filename string) ([]Mutation, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var mutations []Mutation
	err = json.Unmarshal(data, &mutations)
	return mutations, err
}

func applyMutation(m *Mutation) (string, error) {
	// Read the original source file.
	content, err := os.ReadFile(m.File)
	if err != nil {
		return "", err
	}

	// Strategy: Use a line-based approach with position tracking.
	// For complex mutations, we parse the AST and rewrite specific nodes.

	switch m.Type {
	case "arithmetic_operator", "relational_operator", "logical_operator", "assignment_operator", "inc_dec":
		return applyOperatorMutation(m, string(content))

	case "unary_operator":
		return applyUnaryMutation(m, string(content))

	case "constant":
		return applyConstantMutation(m, string(content))

	case "return_value":
		return applyReturnValueMutation(m, string(content))

	case "statement_removal":
		return applyStatementRemoval(m, string(content))

	case "condition_negation":
		return applyConditionNegation(m, string(content))

	case "else_removal":
		return applyElseRemoval(m, string(content))

	default:
		return "", fmt.Errorf("unknown mutation type: %s", m.Type)
	}
}

// applyOperatorMutation replaces an operator at the specified location.
func applyOperatorMutation(m *Mutation, content string) (string, error) {
	return replaceAtPosition(content, m.Line, m.Column, m.Original, m.Mutated)
}

// applyUnaryMutation handles unary operator mutations.
func applyUnaryMutation(m *Mutation, content string) (string, error) {
	if m.Mutated == "(removed)" {
		// Remove the unary operator (e.g., ! or -).
		return removeAtPosition(content, m.Line, m.Column, m.Original)
	}
	return replaceAtPosition(content, m.Line, m.Column, m.Original, m.Mutated)
}

// applyConstantMutation replaces a constant value.
func applyConstantMutation(m *Mutation, content string) (string, error) {
	return replaceAtPosition(content, m.Line, m.Column, m.Original, m.Mutated)
}

// applyReturnValueMutation changes return values.
func applyReturnValueMutation(m *Mutation, content string) (string, error) {
	return replaceAtPosition(content, m.Line, m.Column, m.Original, m.Mutated)
}

// applyStatementRemoval removes or comments out a statement.
func applyStatementRemoval(m *Mutation, content string) (string, error) {
	// Comment out the entire line.
	lines := strings.Split(content, "\n")
	if m.Line < 1 || m.Line > len(lines) {
		return "", fmt.Errorf("line %d out of range", m.Line)
	}

	targetLine := m.Line - 1 // 0-indexed.
	lines[targetLine] = "// MUTANT: " + lines[targetLine]

	return strings.Join(lines, "\n"), nil
}

// applyConditionNegation negates an if condition.
func applyConditionNegation(m *Mutation, content string) (string, error) {
	// Find the condition and add negation.
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, m.File, content, 0)
	if err != nil {
		return "", err
	}

	// Find the if statement at the specified line.
	var targetStmt *ast.IfStmt
	ast.Inspect(node, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			pos := fset.Position(ifStmt.Pos())
			if pos.Line == m.Line {
				targetStmt = ifStmt
				return false
			}
		}
		return true
	})

	if targetStmt == nil {
		return "", fmt.Errorf("if statement not found at line %d", m.Line)
	}

	// Wrap condition in negation.
	targetStmt.Cond = &ast.UnaryExpr{
		Op: token.NOT,
		X:  targetStmt.Cond,
	}

	// Format and return.
	return formatAST(fset, node)
}

// applyElseRemoval removes else branch.
func applyElseRemoval(m *Mutation, content string) (string, error) {
	// Parse and remove else branch from if statement.
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, m.File, content, 0)
	if err != nil {
		return "", err
	}

	// Find the if statement at the specified line.
	var targetStmt *ast.IfStmt
	ast.Inspect(node, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			pos := fset.Position(ifStmt.Pos())
			if pos.Line == m.Line {
				targetStmt = ifStmt
				return false
			}
		}
		return true
	})

	if targetStmt == nil {
		return "", fmt.Errorf("if statement not found at line %d", m.Line)
	}

	// Remove else branch.
	targetStmt.Else = nil

	// Format and return.
	return formatAST(fset, node)
}

// replaceAtPosition replaces text at a specific line and column.
func replaceAtPosition(content string, line, col int, original, replacement string) (string, error) {
	lines := strings.Split(content, "\n")
	if line < 1 || line > len(lines) {
		return "", fmt.Errorf("line %d out of range", line)
	}

	targetLine := line - 1 // 0-indexed.
	lineContent := lines[targetLine]

	// Find the original text in the line.
	// Note: col is 1-indexed in go/token.
	index := strings.Index(lineContent, original)
	if index == -1 {
		// Fallback: try finding it anywhere in the line.
		index = strings.Index(lineContent, original)
		if index == -1 {
			return "", fmt.Errorf("original text %q not found in line %d", original, line)
		}
	}

	// Replace.
	lines[targetLine] = lineContent[:index] + replacement + lineContent[index+len(original):]

	return strings.Join(lines, "\n"), nil
}

// removeAtPosition removes text at a specific location.
func removeAtPosition(content string, line, col int, text string) (string, error) {
	return replaceAtPosition(content, line, col, text, "")
}

// formatAST formats an AST node back to Go source code.
func formatAST(fset *token.FileSet, node ast.Node) (string, error) {
	var buf strings.Builder
	err := format.Node(&buf, fset, node)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// verifyCode checks if the code is syntactically valid.
func verifyCode(code string) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "mutated.go", code, 0)
	return err
}
