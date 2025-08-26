package kino_test

import (
	"testing"

	"github.com/calumari/kino"
	"github.com/stretchr/testify/require"
)

func TestParseMask(t *testing.T) {
	t.Run("double comma error", func(t *testing.T) {
		_, err := kino.ParseMask("a,,b")
		require.Error(t, err)
	})

	t.Run("duplicate field error", func(t *testing.T) {
		_, err := kino.ParseMask("a,a")
		require.Error(t, err)
	})

	t.Run("empty subtree error", func(t *testing.T) {
		_, err := kino.ParseMask("a:()")
		require.Error(t, err)
	})

	t.Run("a,-b positive mode", func(t *testing.T) {
		m, err := kino.ParseMask("a,-b")
		require.NoError(t, err)
		require.Equal(t, kino.Positive, m.Mode)
	})

	t.Run("-a,-b negative mode", func(t *testing.T) {
		m, err := kino.ParseMask("-a,-b")
		require.NoError(t, err)
		require.Equal(t, kino.Negative, m.Mode)
	})

	t.Run("a,-b,c:(d,-e),-z:(x) structure parsed", func(t *testing.T) {
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

	t.Run("z:(-x) structure parsed", func(t *testing.T) {
		m, err := kino.ParseMask("z:(-x)")
		require.NoError(t, err)
		require.Equal(t, kino.Positive, m.Mode)
		require.Len(t, m.Fields, 1)
		zNode := m.Fields["z"]
		require.Equal(t, kino.Positive, zNode.Op)
		require.NotNil(t, zNode.Children)
		require.Equal(t, kino.Negative, zNode.Children.Fields["x"].Op)
	})
}
