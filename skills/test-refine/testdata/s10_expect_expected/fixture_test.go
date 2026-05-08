// Fixture for S10 (expect-the-expected: want derived from same path
// as got).
package fixture

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func normalize(s string) string { return s }

// TP: both args are CallExprs to the same function. exprText match.
func TestExpectExpectedSameFn(t *testing.T) {
	input := "X"
	require.Equal(t, normalize(input), normalize(input))
}

// FP: distinct functions on each side. Should not flag S10.
func TestExpectExpectedDistinctFn(t *testing.T) {
	input := "X"
	require.Equal(t, "x", normalize(input))
}

// FP: simple var-vs-var equality.
func TestExpectExpectedVarOnly(t *testing.T) {
	want := "x"
	got := "x"
	require.Equal(t, want, got)
}
