// score reads JSON-lines findings on stdin and emits a sorted JSON array
// on stdout, with each finding annotated with a composite priority score.
//
// priority = w_risk     * risk_score(file_path)
//          + w_severity * severity(smell_id)
//          + w_gap      * branch_gap(function_under_test)
//
// Defaults: w_risk=0.5, w_severity=0.3, w_gap=0.2.
// Override with --weights risk=0.6,severity=0.3,gap=0.1
//
// Coverage data (optional) is read from --coverage <go-cover-func.txt>
// (output of `go tool cover -func=cov.out`). When absent, gap is 0 for
// every finding.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Finding struct {
	File              string  `json:"file"`
	Line              int     `json:"line"`
	TestName          string  `json:"test_name,omitempty"`
	Smell             string  `json:"smell"`
	Severity          string  `json:"severity"`
	Message           string  `json:"message"`
	Confidence        float64 `json:"confidence,omitempty"`
	FixKind           string  `json:"fix_kind,omitempty"`
	FunctionUnderTest string  `json:"function_under_test,omitempty"`
	Context           string  `json:"context,omitempty"`
	Suggestion        string  `json:"suggestion,omitempty"`
	Priority          float64 `json:"priority,omitempty"`
	Risk              float64 `json:"risk,omitempty"`
	SevScore          float64 `json:"severity_score,omitempty"`
	Gap               float64 `json:"gap,omitempty"`
}

// anyNonzero returns true iff some f in findings has fn(f) > 0.
func anyNonzero(findings []Finding, fn func(Finding) float64) bool {
	for _, f := range findings {
		if fn(f) > 0 {
			return true
		}
	}
	return false
}

func main() {
	weightsFlag := flag.String("weights", "risk=0.5,severity=0.3,gap=0.2",
		"weights for composite score: risk=...,severity=...,gap=...")
	coverPath := flag.String("coverage", "",
		"path to `go tool cover -func=cov.out` output for branch_gap signal")
	flag.Parse()

	wRisk, wSev, wGap := parseWeights(*weightsFlag)
	cov := loadCoverage(*coverPath)

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1024*1024), 1024*1024*8)
	var findings []Finding
	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		var f Finding
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			fmt.Fprintf(os.Stderr, "skip invalid finding: %v\n", err)
			continue
		}
		f.Risk = riskScore(f.File)
		f.SevScore = severityScore(f.Severity)
		f.Gap = coverageGap(cov, f.FunctionUnderTest)
		findings = append(findings, f)
	}

	// If no finding has a usable branch_gap (coverage data missing or
	// 0 across the board — common on itest packages with build tags),
	// renormalize the formula so risk and severity drive ordering
	// instead of collapsing every finding into the same priority bucket.
	if !anyNonzero(findings, func(f Finding) float64 { return f.Gap }) {
		fmt.Fprintln(os.Stderr,
			"warn: branch_gap is zero for all findings; renormalizing weights")
		denom := wRisk + wSev
		if denom > 0 {
			wRisk /= denom
			wSev /= denom
			wGap = 0
		}
	}

	for i := range findings {
		f := &findings[i]
		f.Priority = wRisk*f.Risk + wSev*f.SevScore + wGap*f.Gap
		// Confidence (when set on the finding) attenuates priority so
		// low-confidence detections rank below high-confidence ones at
		// equal raw priority. A confidence of 0 effectively suppresses
		// the finding — caller can choose to drop those.
		if f.Confidence > 0 && f.Confidence < 1 {
			f.Priority *= f.Confidence
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		return findings[i].Priority > findings[j].Priority
	})

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(findings)
}

func parseWeights(s string) (risk, sev, gap float64) {
	risk, sev, gap = 0.5, 0.3, 0.2
	for _, part := range strings.Split(s, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		v, err := strconv.ParseFloat(kv[1], 64)
		if err != nil {
			continue
		}
		switch kv[0] {
		case "risk":
			risk = v
		case "severity", "sev":
			sev = v
		case "gap":
			gap = v
		}
	}
	return risk, sev, gap
}

// riskScore is a heuristic in [0.2, 1.0] based on path keywords.
func riskScore(path string) float64 {
	p := strings.ToLower(path)
	criticalRe := regexp.MustCompile(
		`(consensus|channel|commit|payment|crypto|sign|verify|wallet|htlc|invoice|onion|sphinx|musig|taproot)`,
	)
	switch {
	case criticalRe.MatchString(p):
		return 1.0
	case strings.Contains(p, "/internal/"):
		return 0.7
	case strings.HasPrefix(p, "internal/"):
		return 0.7
	case strings.Contains(p, "/cmd/"):
		return 0.4
	case strings.HasPrefix(p, "cmd/"):
		return 0.4
	case strings.Contains(p, "/test/") || strings.Contains(p, "/testing/"):
		return 0.2
	}
	return 0.5
}

func severityScore(s string) float64 {
	switch strings.ToUpper(s) {
	case "H", "HIGH":
		return 1.0
	case "M", "MEDIUM", "MED":
		return 0.6
	case "L", "LOW":
		return 0.3
	}
	return 0.3
}

// loadCoverage parses the `go tool cover -func` output into a map of
// "package.Func" -> uncovered fraction (0..1).
//
// Sample line:
//   github.com/foo/bar/internal/wallet.go:42:    CalculateFee    78.6%
func loadCoverage(path string) map[string]float64 {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: cannot read coverage: %v\n", err)
		return nil
	}
	defer f.Close()

	out := map[string]float64{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		pct, err := parsePercent(fields[len(fields)-1])
		if err != nil {
			continue
		}
		fnName := fields[len(fields)-2]
		// fields[0] is "path:line:" — extract package-relative path.
		loc := strings.TrimSuffix(fields[0], ":")
		fileColon := strings.Index(loc, ":")
		if fileColon < 0 {
			continue
		}
		filePath := loc[:fileColon]

		// Compute "uncovered" as 1 - covered.
		gap := 1.0 - pct/100.0

		// Index by both bare name and pkg-qualified name.
		out[fnName] = gap
		out[filePath+"."+fnName] = gap
	}
	return out
}

func parsePercent(s string) (float64, error) {
	s = strings.TrimSuffix(s, "%")
	return strconv.ParseFloat(s, 64)
}

func coverageGap(cov map[string]float64, fnUnderTest string) float64 {
	if cov == nil || fnUnderTest == "" {
		return 0
	}
	if v, ok := cov[fnUnderTest]; ok {
		return v
	}
	// Try the bare function name (strip pkg or receiver).
	bare := fnUnderTest
	if i := strings.LastIndex(bare, "."); i >= 0 {
		bare = bare[i+1:]
	}
	if v, ok := cov[bare]; ok {
		return v
	}
	return 0
}
