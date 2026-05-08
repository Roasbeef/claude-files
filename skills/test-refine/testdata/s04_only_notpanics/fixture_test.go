// Fixture for S04 (only NotPanics) and adjacent recover-only patterns.
package fixture

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Named without a "must" prefix so isHelperAssertName doesn't classify
// it as a helper assertion call (which would mask the S04 signal).
func doSomething() {}

// TP: only assertion is NotPanics — behaviour unverified beyond
// "did not crash".
func TestOnlyNotPanics(t *testing.T) {
	require.NotPanics(t, func() { doSomething() })
}

// TP: defer recover() with no other assertion.
func TestOnlyDeferRecover(t *testing.T) {
	defer func() { _ = recover() }()
	doSomething()
}

// FP: NotPanics is one of multiple assertions, others are real.
// Detector must not fire — the test does verify behaviour, just also
// guards against panic.
func TestNotPanicsPlusReal(t *testing.T) {
	x := 5
	require.NotPanics(t, func() { _ = x + 1 })
	require.Equal(t, 5, x)
}

// FP: recover() inside a multi-assertion test.
func TestRecoverPlusReal(t *testing.T) {
	defer func() { _ = recover() }()
	x := 5
	require.Equal(t, 5, x)
}
