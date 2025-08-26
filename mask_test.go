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
