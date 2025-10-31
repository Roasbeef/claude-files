package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Mutation represents a mutation descriptor.
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

// MutationResult represents the result of testing a mutation.
type MutationResult struct {
	MutationID  string  `json:"mutation_id"`
	Status      string  `json:"status"` // killed, survived, timeout, error
	Duration    float64 `json:"duration_ms"`
	TestOutput  string  `json:"test_output"`
	Description string  `json:"description"`
	File        string  `json:"file"`
	Line        int     `json:"line"`
}

var (
	mutationFile = flag.String("mutation-file", "", "JSON file containing mutations")
	mutationID   = flag.String("mutation-id", "", "Specific mutation ID to test (e.g., M0)")
	packagePath  = flag.String("package", ".", "Package to test (e.g., ./internal/wallet)")
	testTimeout  = flag.Duration("timeout", 60*time.Second, "Timeout for test execution")
	output       = flag.String("output", "", "Output file for result JSON (default: stdout)")
	verbose      = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Parse()

	if *mutationFile == "" || *mutationID == "" {
		fmt.Fprintln(os.Stderr, "Error: --mutation-file and --mutation-id are required")
		flag.Usage()
		os.Exit(1)
	}

	// Load mutations.
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

	// Run mutation test.
	result := runMutationTest(mutation)

	// Output result.
	outputResult(result)
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

func runMutationTest(mutation *Mutation) MutationResult {
	start := time.Now()

	result := MutationResult{
		MutationID:  mutation.ID,
		Description: mutation.Description,
		File:        mutation.File,
		Line:        mutation.Line,
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Testing mutation %s: %s\n", mutation.ID, mutation.Description)
	}

	// Step 1: Backup original file.
	originalContent, err := os.ReadFile(mutation.File)
	if err != nil {
		result.Status = "error"
		result.TestOutput = fmt.Sprintf("Failed to read original file: %v", err)
		result.Duration = time.Since(start).Seconds() * 1000
		return result
	}

	backupFile := mutation.File + ".mutation_backup"
	err = os.WriteFile(backupFile, originalContent, 0644)
	if err != nil {
		result.Status = "error"
		result.TestOutput = fmt.Sprintf("Failed to create backup: %v", err)
		result.Duration = time.Since(start).Seconds() * 1000
		return result
	}

	defer func() {
		// Always restore original file.
		os.WriteFile(mutation.File, originalContent, 0644)
		os.Remove(backupFile)
	}()

	// Step 2: Apply mutation.
	mutatedCode, err := applyMutationToCode(mutation, string(originalContent))
	if err != nil {
		result.Status = "error"
		result.TestOutput = fmt.Sprintf("Failed to apply mutation: %v", err)
		result.Duration = time.Since(start).Seconds() * 1000
		return result
	}

	// Write mutated code.
	err = os.WriteFile(mutation.File, []byte(mutatedCode), 0644)
	if err != nil {
		result.Status = "error"
		result.TestOutput = fmt.Sprintf("Failed to write mutated code: %v", err)
		result.Duration = time.Since(start).Seconds() * 1000
		return result
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Applied mutation to %s\n", mutation.File)
	}

	// Step 3: Run tests.
	testPassed, testOutput, testErr := runTests(*packagePath, *testTimeout)

	result.Duration = time.Since(start).Seconds() * 1000
	result.TestOutput = testOutput

	// Step 4: Determine result.
	if testErr != nil {
		if strings.Contains(testErr.Error(), "timeout") {
			result.Status = "timeout"
		} else {
			// Test command failed (mutant killed).
			result.Status = "killed"
		}
	} else if testPassed {
		// Tests passed with mutation (mutant survived).
		result.Status = "survived"
	} else {
		// Tests failed (mutant killed).
		result.Status = "killed"
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Result: %s (%.2fms)\n", result.Status, result.Duration)
	}

	return result
}

func applyMutationToCode(mutation *Mutation, content string) (string, error) {
	// Simple line-based mutation application.
	lines := strings.Split(content, "\n")

	if mutation.Line < 1 || mutation.Line > len(lines) {
		return "", fmt.Errorf("line %d out of range (file has %d lines)", mutation.Line, len(lines))
	}

	targetLine := mutation.Line - 1 // 0-indexed.

	// Apply mutation based on type.
	switch mutation.Type {
	case "arithmetic_operator", "relational_operator", "logical_operator", "assignment_operator", "inc_dec":
		// Replace operator.
		lines[targetLine] = strings.Replace(lines[targetLine], mutation.Original, mutation.Mutated, 1)

	case "unary_operator":
		if mutation.Mutated == "(removed)" {
			// Remove unary operator.
			lines[targetLine] = strings.Replace(lines[targetLine], mutation.Original, "", 1)
		} else {
			lines[targetLine] = strings.Replace(lines[targetLine], mutation.Original, mutation.Mutated, 1)
		}

	case "constant":
		// Replace constant value.
		lines[targetLine] = strings.Replace(lines[targetLine], mutation.Original, mutation.Mutated, 1)

	case "return_value":
		// Replace return value.
		lines[targetLine] = strings.Replace(lines[targetLine], mutation.Original, mutation.Mutated, 1)

	case "statement_removal":
		// Comment out the line.
		lines[targetLine] = "// MUTANT_REMOVED: " + lines[targetLine]

	case "condition_negation":
		// Add negation to if condition.
		// This is a simplified version - full implementation would parse AST.
		ifIndex := strings.Index(lines[targetLine], "if ")
		if ifIndex != -1 {
			// Find the condition part.
			condStart := ifIndex + 3
			braceIndex := strings.Index(lines[targetLine][condStart:], "{")
			if braceIndex != -1 {
				condition := strings.TrimSpace(lines[targetLine][condStart : condStart+braceIndex])
				lines[targetLine] = lines[targetLine][:condStart] + "!(" + condition + ") " + lines[targetLine][condStart+braceIndex:]
			}
		}

	case "else_removal":
		// Remove else branch (simplified - just comment it out).
		if strings.Contains(lines[targetLine], "else") {
			lines[targetLine] = "// MUTANT_REMOVED_ELSE: " + lines[targetLine]
		}

	default:
		return "", fmt.Errorf("unknown mutation type: %s", mutation.Type)
	}

	return strings.Join(lines, "\n"), nil
}

func runTests(packagePath string, timeout time.Duration) (bool, string, error) {
	// Run go test with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-v", packagePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout and stderr.
	output := stdout.String() + stderr.String()

	if err != nil {
		// Check if it's a timeout.
		if ctx.Err() != nil {
			return false, output, fmt.Errorf("timeout after %v", timeout)
		}

		// Test failed (which is good - mutant killed).
		return false, output, nil
	}

	// Test passed (mutant survived).
	return true, output, nil
}

func outputResult(result MutationResult) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling result: %v\n", err)
		os.Exit(1)
	}

	if *output == "" {
		fmt.Println(string(data))
	} else {
		// Create output directory if needed.
		dir := filepath.Dir(*output)
		os.MkdirAll(dir, 0755)

		err := os.WriteFile(*output, data, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}

		if *verbose {
			fmt.Fprintf(os.Stderr, "Result written to %s\n", *output)
		}
	}
}
