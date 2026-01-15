package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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

var (
	mutationFile = flag.String("mutation-file", "", "JSON file containing mutations")
	packagePath  = flag.String("package", ".", "Package to test")
	outputDir    = flag.String("output-dir", "mutation-results", "Directory for result JSON files")
	parallel     = flag.Int("parallel", 1, "MUST be 1: mutations modify files in-place (not safe for parallel)")
	skipImports  = flag.Bool("skip-imports", true, "Skip mutations in import statements (lines < 20)")
	minLine      = flag.Int("min-line", 0, "Only test mutations at or after this line")
	maxLine      = flag.Int("max-line", 0, "Only test mutations at or before this line (0 = no limit)")
	verbose      = flag.Bool("verbose", false, "Verbose output")
	timeout      = flag.Duration("timeout", 60*time.Second, "Timeout per mutation test")
)

func main() {
	flag.Parse()

	if *mutationFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --mutation-file is required")
		flag.Usage()
		os.Exit(1)
	}

	// SAFETY: Force sequential execution.
	if *parallel != 1 {
		fmt.Fprintln(os.Stderr, "WARNING: --parallel must be 1 because mutations modify files in-place.")
		fmt.Fprintln(os.Stderr, "Parallel execution would cause race conditions and corrupted results.")
		fmt.Fprintln(os.Stderr, "Setting parallel=1 for safety.")
		*parallel = 1
	}

	// Create output directory.
	err := os.MkdirAll(*outputDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Load mutations.
	mutations, err := loadMutations(*mutationFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading mutations: %v\n", err)
		os.Exit(1)
	}

	// Filter mutations.
	filtered := filterMutations(mutations)

	fmt.Printf("Loaded %d mutations, testing %d mutations (filtered: %d)\n",
		len(mutations), len(filtered), len(mutations)-len(filtered))

	if len(filtered) == 0 {
		fmt.Println("No mutations to test")
		return
	}

	// Run mutations in parallel.
	start := time.Now()
	results := runMutationsParallel(filtered)
	duration := time.Since(start)

	// Summary.
	killed := 0
	survived := 0
	timeouts := 0
	errors := 0

	for _, result := range results {
		switch result.Status {
		case "killed":
			killed++
		case "survived":
			survived++
		case "timeout":
			timeouts++
		case "error":
			errors++
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Mutation Testing Complete")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total:    %d\n", len(filtered))
	fmt.Printf("Killed:   %d (%.1f%%)\n", killed, float64(killed)/float64(len(filtered))*100)
	fmt.Printf("Survived: %d (%.1f%%)\n", survived, float64(survived)/float64(len(filtered))*100)
	fmt.Printf("Timeout:  %d\n", timeouts)
	fmt.Printf("Errors:   %d\n", errors)

	// Mutation score (excludes timeouts).
	testable := len(filtered) - timeouts
	if testable > 0 {
		score := float64(killed) / float64(testable) * 100
		fmt.Printf("\nMutation Score: %.1f%% (%d/%d killed, %d timeouts excluded)\n",
			score, killed, testable, timeouts)
	}

	fmt.Printf("Duration: %v\n", duration.Round(time.Millisecond))
	fmt.Println(strings.Repeat("=", 60))

	// Suggest running parse-results.
	fmt.Printf("\nRun parse-results.sh to generate detailed report:\n")
	fmt.Printf("  ~/.claude/skills/mutation-testing/scripts/parse-results.sh \\\n")
	fmt.Printf("    --results '%s/*.json' \\\n", *outputDir)
	fmt.Printf("    --output mutation-report.json\n")
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

func filterMutations(mutations []Mutation) []Mutation {
	var filtered []Mutation

	for _, m := range mutations {
		// Skip imports if requested.
		if *skipImports && m.Line < 20 {
			continue
		}

		// Apply min/max line filters.
		if *minLine > 0 && m.Line < *minLine {
			continue
		}
		if *maxLine > 0 && m.Line > *maxLine {
			continue
		}

		filtered = append(filtered, m)
	}

	return filtered
}

type mutationResult struct {
	ID     string
	Status string
	Error  error
}

func runMutationsParallel(mutations []Mutation) []mutationResult {
	results := make([]mutationResult, len(mutations))
	var completed atomic.Int32
	total := len(mutations)

	// Create worker pool.
	jobs := make(chan int, len(mutations))
	var wg sync.WaitGroup

	// Start workers.
	for i := 0; i < *parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				mutation := mutations[idx]
				status, err := runSingleMutation(mutation)
				results[idx] = mutationResult{
					ID:     mutation.ID,
					Status: status,
					Error:  err,
				}

				count := completed.Add(1)
				if *verbose || count%10 == 0 || int(count) == total {
					fmt.Printf("[%d/%d] %s: %s\n", count, total, mutation.ID, status)
				}
			}
		}()
	}

	// Queue jobs.
	for i := range mutations {
		jobs <- i
	}
	close(jobs)

	// Wait for completion.
	wg.Wait()

	return results
}

func runSingleMutation(mutation Mutation) (string, error) {
	outputFile := filepath.Join(*outputDir, mutation.ID+".json")

	// Build command - use runtime.Caller to get actual script directory.
	_, filename, _, _ := runtime.Caller(0)
	scriptDir := filepath.Dir(filename)
	runScript := filepath.Join(scriptDir, "run-mutation-test.sh")

	cmd := exec.Command(
		runScript,
		"--mutation-file", *mutationFile,
		"--mutation-id", mutation.ID,
		"--package", *packagePath,
		"--output", outputFile,
		"--timeout", timeout.String(),
	)

	// Run with overall timeout (add buffer).
	ctx, cancel := context.WithTimeout(context.Background(), *timeout+10*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if output file was created with status.
		if data, readErr := os.ReadFile(outputFile); readErr == nil {
			var result struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(data, &result) == nil {
				return result.Status, nil
			}
		}
		return "error", fmt.Errorf("failed to run mutation: %v: %s", err, output)
	}

	// Read result file to get status.
	data, err := os.ReadFile(outputFile)
	if err != nil {
		return "error", fmt.Errorf("failed to read result: %v", err)
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "error", fmt.Errorf("failed to parse result: %v", err)
	}

	return result.Status, nil
}
