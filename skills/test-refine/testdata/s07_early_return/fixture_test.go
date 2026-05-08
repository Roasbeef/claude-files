// Fixture for S07 (early-return bypasses subsequent assertions).
package fixture

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func makeValue() (int, bool) { return 1, true }

// TP: an `if !ok { return }` followed by a real require.X assertion.
// The early return makes the subsequent assertion conditionally
// executed — easy to silently skip a check.
func TestEarlyReturnBypassesAssert(t *testing.T) {
	v, ok := makeValue()
	if !ok {
		return
	}
	require.Equal(t, 1, v)
}

// FP: early return BEFORE any assertion exists — there's no assert
// being bypassed.
func TestEarlyReturnNoAssertAfter(t *testing.T) {
	_, ok := makeValue()
	if !ok {
		return
	}
	// no assertions at all.
	_ = ok
}

// FP: t.Skip is the canonical way to skip; no assert is being silently
// bypassed because Skip marks the test as skipped, not passed.
func TestSkipInsteadOfReturn(t *testing.T) {
	_, ok := makeValue()
	if !ok {
		t.Skip("no value available")
	}
	require.True(t, ok)
}

// FP: early return in a subtest closure — the parent test still runs
// the asserts, the subtest just stops.
func TestSubtestEarlyReturn(t *testing.T) {
	t.Run("inner", func(t *testing.T) {
		_, ok := makeValue()
		if !ok {
			return
		}
		require.True(t, ok)
	})
}
