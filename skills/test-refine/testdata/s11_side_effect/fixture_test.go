// Fixture for S11 (SUT mutates argument; mutation not asserted).
package fixture

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Buf struct{ Data []byte }

func mutate(b *Buf) { b.Data = append(b.Data, 1) }

// TP: mutate() is called with a pointer; nothing reads back from b
// before the test ends. Side effect goes unverified.
func TestMutateNoReadback(t *testing.T) {
	b := &Buf{}
	mutate(b)
}

// FP: pointer passed, then a field of the same name is read back via
// require.Len. Detector must not flag.
func TestMutateWithReadback(t *testing.T) {
	b := &Buf{}
	mutate(b)
	require.Len(t, b.Data, 1)
}

// FP: passing `t` (the testing handle) to a helper. Must not flag —
// `t` is the testing parameter, not a SUT side effect.
func TestPassTestingT(t *testing.T) {
	helperWithT(t)
}

func helperWithT(t *testing.T) { t.Helper() }

// FP: TRUE single-delegation body — exactly one ExprStmt CallExpr.
// The runner exercises the SUT and any readback happens inside it.
// The detector's isSingleDelegation gate is intentionally strict:
// any preceding declaration disqualifies the test, because then the
// declared object's lifecycle becomes the test's responsibility.
func TestSingleDelegation(t *testing.T) {
	runMutationScenario(t)
}

func runMutationScenario(t *testing.T) {
	b := &Buf{}
	mutate(b)
	require.NotEmpty(t, b.Data)
}
