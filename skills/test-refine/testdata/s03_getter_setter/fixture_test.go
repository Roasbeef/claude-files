// Fixture for S03 (getter/setter trivial) detector.
package fixture

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type box struct{ v int }

func (b *box) SetV(x int)   { b.v = x }
func (b *box) GetV() int    { return b.v }
func (b *box) SetVErr(int) error { return nil }

// TP: classic Set then GetV roundtrip with require.Equal.
func TestBoxSetGetEqual(t *testing.T) {
	b := &box{}
	b.SetV(5)
	require.Equal(t, 5, b.GetV())
}

// TP: Set then `if b.GetV() != v { t.Errorf }` form.
func TestBoxSetGetManualFail(t *testing.T) {
	b := &box{}
	b.SetV(7)
	if b.GetV() != 7 {
		t.Errorf("expected 7, got %d", b.GetV())
	}
}

// FP: Set + Get but with a non-trivial computation/assertion in between.
// Detector cannot easily tell — currently it just looks for Set/Get
// pairs, so this MAY false-positive. Document the limitation.
func TestBoxSetGetWithIntermediate(t *testing.T) {
	b := &box{}
	b.SetV(1)
	for range 10 {
		b.SetV(b.GetV() + 1)
	}
	require.Equal(t, 11, b.GetV())
}

// FP: SetVErr returns an error and is checked. Not a trivial Set/Get
// — there's a real assertion on the SUT's error path.
func TestBoxSetVErr(t *testing.T) {
	b := &box{}
	err := b.SetVErr(1)
	require.NoError(t, err)
}

// FP: only a Get, no Set. Not the trivial pattern.
func TestBoxOnlyGet(t *testing.T) {
	b := &box{v: 3}
	require.Equal(t, 3, b.GetV())
}

// FP: only a Set, no Get assertion.
func TestBoxOnlySet(t *testing.T) {
	b := &box{}
	b.SetV(2)
	require.NotNil(t, b)
}
