// Fixture for S02 (tautological assertion) detector.
//
// Each test is annotated with TP (true positive — detector MUST flag)
// or FP (false positive look-alike — detector MUST NOT flag).
package fixture

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TP: same expression on both sides of Equal.
func TestEqualSameVar(t *testing.T) {
	x := 1
	require.Equal(t, x, x)
}

// TP: True called on the literal `true`.
func TestTrueLiteral(t *testing.T) {
	require.True(t, true)
}

// TP: False called on the literal `false`.
func TestFalseLiteral(t *testing.T) {
	require.False(t, false)
}

// TP: identical compound expressions.
func TestEqualSameSelectorChain(t *testing.T) {
	type S struct{ A int }
	s := S{A: 1}
	require.Equal(t, s.A, s.A)
}

// FP: distinct sides — only the literal `1` and `1` match if you
// canonicalise to text. The current detector compares literal text,
// so `1` vs `1` *would* match. Move this to a real-world variable
// pair to exercise the negative case clearly.
func TestEqualDistinctLiterals(t *testing.T) {
	require.Equal(t, 1, 2)
}

// FP: distinct variables.
func TestEqualDistinctVars(t *testing.T) {
	want := 1
	got := 2
	require.Equal(t, want, got)
}

// FP: True with a non-literal (not tautological).
func TestTrueOnExpression(t *testing.T) {
	require.True(t, 1 < 2)
}

// FP: same function called on both sides — NOT a tautology in the
// "detector should flag" sense, this is what S10 catches. S02 must not
// double-flag it. (Detector currently flags exprText equality, which
// IS the same text — so this *would* trip S02. Verify behaviour.)
func TestEqualSameSideEffectingCall(t *testing.T) {
	counter := 0
	inc := func() int { counter++; return counter }
	require.Equal(t, inc(), inc())
}
