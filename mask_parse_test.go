package kino_test

import (
	"testing"

	"github.com/calumari/kino"
	"github.com/stretchr/testify/require"
)

func TestParseMask(t *testing.T) {
	t.Run("error double comma", func(t *testing.T) {
		_, err := kino.ParseMask("a,,b")
		require.Error(t, err)
	})

	t.Run("error duplicate field", func(t *testing.T) {
		_, err := kino.ParseMask("a,a")
		require.Error(t, err)
	})

	t.Run("error empty subtree", func(t *testing.T) {
		_, err := kino.ParseMask("a:()")
		require.Error(t, err)
	})

	t.Run("mode positive", func(t *testing.T) {
		m, err := kino.ParseMask("a,-b")
		require.NoError(t, err)
		require.Equal(t, kino.Positive, m.Mode)
	})

	t.Run("mode negative", func(t *testing.T) {
		m, err := kino.ParseMask("-a,-b")
		require.NoError(t, err)
		require.Equal(t, kino.Negative, m.Mode)
	})

	t.Run("structure a,-b,c:(d,-e),-z:(x)", func(t *testing.T) {
		m, err := kino.ParseMask("a,-b,c:(d,-e),-z:(x)")
		require.NoError(t, err)
		require.Equal(t, kino.Positive, m.Mode)
		require.Len(t, m.Fields, 4)
		require.Equal(t, kino.Positive, m.Fields["a"].Op)
		require.Equal(t, kino.Negative, m.Fields["b"].Op)
		cNode := m.Fields["c"]
		require.NotNil(t, cNode.Children)
		require.Equal(t, kino.Positive, cNode.Children.Fields["d"].Op)
		require.Equal(t, kino.Negative, cNode.Children.Fields["e"].Op)
		zNode := m.Fields["z"]
		require.Equal(t, kino.Negative, zNode.Op)
		require.NotNil(t, zNode.Children)
		require.Equal(t, kino.Positive, zNode.Children.Fields["x"].Op)
	})
}
