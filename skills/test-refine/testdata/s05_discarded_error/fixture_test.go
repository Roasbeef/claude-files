// Fixture for S05 (discarded error from SUT).
package fixture

import (
	"testing"
)

// SAME-PACKAGE callees with known error vs non-error returns.

func errorReturner() error          { return nil }
func nonErrorReturner() (int, bool) { return 1, true }
func multiReturnNoError() (int, string, []byte) {
	return 0, "", nil
}

// TP: same-package fn that returns error, discarded with `_`. Confidence 1.0.
func TestDiscardErrorReturn(t *testing.T) {
	_ = errorReturner()
}

// FP: helper returns (T, U) — neither is error. The detector must NOT flag.
// This was the pattern that produced the original feedback round's noise.
func TestDiscardNonErrorTuple(t *testing.T) {
	x, _ := nonErrorReturner()
	_ = x
}

// FP: triple-return helper, none of them error.
func TestDiscardThreeReturnsNoError(t *testing.T) {
	a, _, _ := multiReturnNoError()
	_ = a
}

// FP: builtin recover() — adopted in S04 path.
func TestDiscardRecoverBuiltin(t *testing.T) {
	defer func() { _ = recover() }()
}

// FP: builtin len() discard.
func TestDiscardLenBuiltin(t *testing.T) {
	s := []int{1}
	_ = len(s)
}
