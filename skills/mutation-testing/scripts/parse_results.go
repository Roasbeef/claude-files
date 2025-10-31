package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MutationResult represents a single mutation test result.
type MutationResult struct {
	MutationID  string  `json:"mutation_id"`
	Status      string  `json:"status"` // killed, survived, timeout, error
	Duration    float64 `json:"duration_ms"`
	TestOutput  string  `json:"test_output"`
	Description string  `json:"description"`
	File        string  `json:"file"`
	Line        int     `json:"line"`
}

// AggregatedResults contains summary statistics and details.
type AggregatedResults struct {
	TotalMutations   int               `json:"total_mutations"`
	Killed           int               `json:"killed"`
	Survived         int               `json:"survived"`
	Timeouts         int               `json:"timeouts"`
	Errors           int               `json:"errors"`
	MutationScore    float64           `json:"mutation_score"`
	TotalDuration    float64           `json:"total_duration_ms"`
	Results          []MutationResult  `json:"results"`
	SurvivorsByFile  map[string]int    `json:"survivors_by_file"`
	SurvivorsByType  map[string]int    `json:"survivors_by_type"`
	Survivors        []MutationResult  `json:"survivors"` // Detailed survivor info.
}

var (
	resultsPattern = flag.String("results", "", "Pattern for result JSON files (e.g., results/*.json)")
	output         = flag.String("output", "", "Output file for aggregated results (default: stdout)")
	verbose        = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Parse()

	if *resultsPattern == "" {
		fmt.Fprintln(os.Stderr, "Error: --results pattern is required")
		flag.Usage()
		os.Exit(1)
	}

	// Find all matching result files.
	files, err := filepath.Glob(*resultsPattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding result files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No result files found matching pattern: %s\n", *resultsPattern)
		os.Exit(1)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Found %d result files\n", len(files))
	}

	// Load and aggregate results.
	aggregated := aggregateResults(files)

	// Output aggregated results.
	outputResults(aggregated)
}

func aggregateResults(files []string) AggregatedResults {
	agg := AggregatedResults{
		Results:         []MutationResult{},
		SurvivorsByFile: make(map[string]int),
		SurvivorsByType: make(map[string]int),
		Survivors:       []MutationResult{},
	}

	for _, file := range files {
		result, err := loadResult(file)
		if err != nil {
			if *verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", file, err)
			}
			continue
		}

		agg.Results = append(agg.Results, result)
		agg.TotalMutations++
		agg.TotalDuration += result.Duration

		// Count by status.
		switch result.Status {
		case "killed":
			agg.Killed++
		case "survived":
			agg.Survived++
			agg.Survivors = append(agg.Survivors, result)

			// Track survivors by file and type.
			agg.SurvivorsByFile[result.File]++

			// Extract type from mutation ID or description.
			mutType := extractMutationType(result)
			agg.SurvivorsByType[mutType]++

		case "timeout":
			agg.Timeouts++
		case "error":
			agg.Errors++
		}
	}

	// Calculate mutation score.
	// Score = killed / (total - timeouts - errors)
	validMutations := agg.TotalMutations - agg.Timeouts - agg.Errors
	if validMutations > 0 {
		agg.MutationScore = float64(agg.Killed) / float64(validMutations) * 100.0
	}

	// Sort survivors by file and line for better reporting.
	sort.Slice(agg.Survivors, func(i, j int) bool {
		if agg.Survivors[i].File != agg.Survivors[j].File {
			return agg.Survivors[i].File < agg.Survivors[j].File
		}
		return agg.Survivors[i].Line < agg.Survivors[j].Line
	})

	return agg
}

func loadResult(filename string) (MutationResult, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return MutationResult{}, err
	}

	var result MutationResult
	err = json.Unmarshal(data, &result)
	return result, err
}

func extractMutationType(result MutationResult) string {
	// Try to extract type from description.
	desc := strings.ToLower(result.Description)

	if strings.Contains(desc, "boundary") || strings.Contains(desc, ">=") || strings.Contains(desc, "<=") {
		return "boundary_condition"
	}
	if strings.Contains(desc, "arithmetic") || strings.Contains(desc, "+") || strings.Contains(desc, "-") {
		return "arithmetic_operator"
	}
	if strings.Contains(desc, "logical") || strings.Contains(desc, "&&") || strings.Contains(desc, "||") {
		return "logical_operator"
	}
	if strings.Contains(desc, "negation") || strings.Contains(desc, "!") {
		return "negation"
	}
	if strings.Contains(desc, "return") {
		return "return_value"
	}
	if strings.Contains(desc, "remove") {
		return "statement_removal"
	}
	if strings.Contains(desc, "constant") || strings.Contains(desc, "0") || strings.Contains(desc, "1") {
		return "constant"
	}

	return "other"
}

func outputResults(agg AggregatedResults) {
	data, err := json.MarshalIndent(agg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling results: %v\n", err)
		os.Exit(1)
	}

	if *output == "" {
		fmt.Println(string(data))

		// Also print summary to stderr if verbose.
		if *verbose {
			printSummary(agg)
		}
	} else {
		// Create output directory if needed.
		dir := filepath.Dir(*output)
		os.MkdirAll(dir, 0755)

		err := os.WriteFile(*output, data, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Results written to %s\n", *output)
		printSummary(agg)
	}
}

func printSummary(agg AggregatedResults) {
	fmt.Fprintln(os.Stderr, "\n=== Mutation Testing Summary ===")
	fmt.Fprintf(os.Stderr, "Total Mutations: %d\n", agg.TotalMutations)
	fmt.Fprintf(os.Stderr, "Killed: %d (%.1f%%)\n", agg.Killed, float64(agg.Killed)/float64(agg.TotalMutations)*100)
	fmt.Fprintf(os.Stderr, "Survived: %d (%.1f%%)\n", agg.Survived, float64(agg.Survived)/float64(agg.TotalMutations)*100)
	fmt.Fprintf(os.Stderr, "Timeouts: %d\n", agg.Timeouts)
	fmt.Fprintf(os.Stderr, "Errors: %d\n", agg.Errors)
	fmt.Fprintf(os.Stderr, "\nMutation Score: %.1f%%\n", agg.MutationScore)
	fmt.Fprintf(os.Stderr, "Total Duration: %.2f seconds\n", agg.TotalDuration/1000.0)

	if len(agg.Survivors) > 0 {
		fmt.Fprintln(os.Stderr, "\n=== Surviving Mutants ===")
		for _, survivor := range agg.Survivors {
			fmt.Fprintf(os.Stderr, "- %s:%d - %s (%s)\n",
				survivor.File, survivor.Line, survivor.Description, survivor.MutationID)
		}

		fmt.Fprintln(os.Stderr, "\n=== Survivors by Type ===")
		for mutType, count := range agg.SurvivorsByType {
			fmt.Fprintf(os.Stderr, "- %s: %d\n", mutType, count)
		}
	}

	// Quality assessment.
	fmt.Fprintln(os.Stderr, "\n=== Quality Assessment ===")
	if agg.MutationScore >= 90 {
		fmt.Fprintln(os.Stderr, "✓ EXCELLENT: Very thorough test suite")
	} else if agg.MutationScore >= 80 {
		fmt.Fprintln(os.Stderr, "✓ GOOD: Solid test coverage with minor gaps")
	} else if agg.MutationScore >= 70 {
		fmt.Fprintln(os.Stderr, "⚠ FAIR: Tests cover main paths but miss edges")
	} else {
		fmt.Fprintln(os.Stderr, "✗ POOR: Significant test gaps, add more tests")
	}
}
