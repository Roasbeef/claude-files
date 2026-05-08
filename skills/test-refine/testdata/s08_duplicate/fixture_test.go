// Fixture for S08 (duplicate test body) detector.
package fixture

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TP: byte-identical body — clearly redundant.
func TestDupeA(t *testing.T) {
	x := 1
	require.Equal(t, 1, x)
}

func TestDupeB(t *testing.T) {
	x := 1
	require.Equal(t, 1, x)
}

// FP: structurally identical but distinct local var name. Detector
// normalises locals, so this WILL flag — that's TP, but kept here to
// document the canonicalisation behaviour.
func TestDupeRenamedC(t *testing.T) {
	y := 1
	require.Equal(t, 1, y)
}

// Restart-variant pattern: both delegate to the same runner with
// different constant arguments. Demoted to S08-VARIANT (low severity).
func runScenario(t *testing.T, mode int) {
	require.Equal(t, mode, mode)
}

const (
	modeA = 1
	modeB = 2
)

func TestRunnerVariantA(t *testing.T) { runScenario(t, modeA) }
func TestRunnerVariantB(t *testing.T) { runScenario(t, modeB) }

// FP: distinct bodies, no overlap.
func TestUnique(t *testing.T) {
	want := "hello"
	require.NotEmpty(t, want)
}
