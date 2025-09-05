package kino_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/calumari/kino"
)

func TestMask_String(t *testing.T) {
	t.Run("deterministic ordering consistent", func(t *testing.T) {
		m := maskPositive(map[string]*kino.Node{
			"c": nodePos(maskPositive(map[string]*kino.Node{"d": {Op: kino.Positive}})),
			"a": {Op: kino.Positive},
			"b": {Op: kino.Negative},
		})
		s1 := m.String()
		s2 := m.String()
		require.Equal(t, s1, s2)
		require.Equal(t, "a,-b,c:(d)", s1)
	})
}

func TestMask_Overlay(t *testing.T) {
	t.Run("nil receiver adopts other", func(t *testing.T) {
		var base *kino.Mask
		overlay := maskPositive(map[string]*kino.Node{"a": {Op: kino.Positive}})
		got := base.Overlay(overlay)
		require.NotNil(t, got)
		require.Len(t, got.Fields, 1)
		// Ensure deep copy (mutate original overlay after)
		overlay.Fields["a"].Op = kino.Negative
		require.Equal(t, kino.Positive, got.Fields["a"].Op)
	})

	t.Run("nil other returns clone", func(t *testing.T) {
		base := maskPositive(map[string]*kino.Node{"a": {Op: kino.Negative}}) // unusual but allowed
		var other *kino.Mask
		got := base.Overlay(other)
		require.NotNil(t, got)
		require.NotSame(t, base, got)
		require.Equal(t, base.Fields["a"].Op, got.Fields["a"].Op)
		base.Fields["a"].Op = kino.Positive
		require.Equal(t, kino.Negative, got.Fields["a"].Op)
	})

	t.Run("disjoint adds missing fields", func(t *testing.T) {
		base := maskPositive(map[string]*kino.Node{"a": {Op: kino.Positive}})
		overlay := maskPositive(map[string]*kino.Node{"b": {Op: kino.Negative}})
		got := base.Overlay(overlay)
		require.ElementsMatch(t, []string{"a", "b"}, keys(got))
		require.Equal(t, kino.Positive, got.Fields["a"].Op)
		require.Equal(t, kino.Negative, got.Fields["b"].Op)
	})

	t.Run("overlapping preserves base op", func(t *testing.T) {
		base := maskPositive(map[string]*kino.Node{"a": {Op: kino.Positive}})
		overlay := maskPositive(map[string]*kino.Node{"a": {Op: kino.Negative}})
		got := base.Overlay(overlay)
		require.Equal(t, kino.Positive, got.Fields["a"].Op)
	})

	t.Run("overlay child with no children ignored", func(t *testing.T) {
		base := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Negative /* no children */},
		})
		overlay := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Positive, Children: maskPositive(map[string]*kino.Node{
				"x": {Op: kino.Positive},
			})},
		})
		got := base.Overlay(overlay)
		require.Equal(t, kino.Negative, got.Fields["a"].Op)
		require.NotNil(t, got.Fields["a"].Children)
		require.ElementsMatch(t, []string{"x"}, keys(got.Fields["a"].Children))
		require.Equal(t, kino.Positive, got.Fields["a"].Children.Fields["x"].Op)
	})

	t.Run("children merged recursively", func(t *testing.T) {
		base := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Positive, Children: maskPositive(map[string]*kino.Node{
				"x": {Op: kino.Positive},
			})},
		})
		overlay := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Negative, Children: maskPositive(map[string]*kino.Node{
				"y": {Op: kino.Negative},
			})},
		})
		got := base.Overlay(overlay)
		// base op preserved, children union
		require.Equal(t, kino.Positive, got.Fields["a"].Op)
		child := got.Fields["a"].Children
		require.NotNil(t, child)
		require.ElementsMatch(t, []string{"x", "y"}, keys(child))
		// x from base positive, y from overlay negative
		require.Equal(t, kino.Positive, child.Fields["x"].Op)
		require.Equal(t, kino.Negative, child.Fields["y"].Op)
	})

	t.Run("mode recomputed negative only children", func(t *testing.T) {
		base := maskPositive(map[string]*kino.Node{})
		overlay := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Negative},
			"b": {Op: kino.Negative},
		})
		got := base.Overlay(overlay)
		require.Equal(t, kino.Negative, got.Mode)
	})

	t.Run("mode recomputed mixed children positive", func(t *testing.T) {
		base := maskPositive(map[string]*kino.Node{})
		overlay := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Negative},
			"b": {Op: kino.Positive},
		})
		got := base.Overlay(overlay)
		require.Equal(t, kino.Positive, got.Mode)
	})

	t.Run("empty both yields empty positive", func(t *testing.T) {
		got := (&kino.Mask{}).Overlay(&kino.Mask{})
		require.NotNil(t, got)
		require.Len(t, got.Fields, 0)
		require.Equal(t, kino.Positive, got.Mode)
	})

	t.Run("existing child nil overlay child ignored", func(t *testing.T) {
		base := maskPositive(map[string]*kino.Node{"a": {Op: kino.Positive}})
		overlay := maskPositive(map[string]*kino.Node{"a": {Op: kino.Negative /* no children */}})
		got := base.Overlay(overlay)
		require.Equal(t, kino.Positive, got.Fields["a"].Op)
		require.Nil(t, got.Fields["a"].Children)
	})

	t.Run("deep immutability modifications don't leak", func(t *testing.T) {
		base := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Positive, Children: maskPositive(map[string]*kino.Node{"x": {Op: kino.Positive}})},
		})
		overlay := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Negative, Children: maskPositive(map[string]*kino.Node{"y": {Op: kino.Negative}})},
		})
		got := base.Overlay(overlay)
		// Mutate originals after merge
		base.Fields["a"].Children.Fields["x"].Op = kino.Negative
		overlay.Fields["a"].Children.Fields["y"].Op = kino.Positive
		// Result stays stable
		child := got.Fields["a"].Children
		require.Equal(t, kino.Positive, child.Fields["x"].Op)
		require.Equal(t, kino.Negative, child.Fields["y"].Op)
	})
}

func keys(m *kino.Mask) []string {
	res := make([]string, 0, len(m.Fields))
	for k := range m.Fields {
		res = append(res, k)
	}
	return res
}
