package mongo

import (
	"testing"

	"github.com/calumari/kino"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestToProjection(t *testing.T) {
	t.Run("nil mask empty projection", func(t *testing.T) {
		require.Equal(t, bson.D{}, Project(nil))
	})

	t.Run("empty mask empty projection", func(t *testing.T) {
		require.Equal(t, bson.D{}, Project(nil))
		require.Equal(t, bson.D{}, Project(&kino.Mask{}))
	})

	t.Run("a,c:(d) positive simple projection", func(t *testing.T) {
		want := bson.D{
			{Key: "a", Value: 1},
			{Key: "c.d", Value: 1},
		}

		m, err := kino.ParseMask("a,c:(d)")
		require.NoError(t, err)
		require.ElementsMatch(t, want, Project(m))
	})

	t.Run("-a,-b negative simple excludes projection", func(t *testing.T) {
		want := bson.D{
			{Key: "a", Value: 0},
			{Key: "b", Value: 0},
		}

		m, err := kino.ParseMask("-a,-b")
		require.NoError(t, err)
		require.ElementsMatch(t, want, Project(m))
	})

	t.Run("-z:(x) override projection", func(t *testing.T) {
		want := bson.D{
			{Key: "z.x", Value: 1},
		}

		m, err := kino.ParseMask("-z:(x)")
		require.NoError(t, err)
		require.ElementsMatch(t, want, Project(m))
	})

	t.Run("a,-b,c:(d,-e),-z:(x) mixed projection", func(t *testing.T) {
		want := bson.D{
			{Key: "a", Value: 1},
			{Key: "c.d", Value: 1},
			{Key: "z.x", Value: 1},
		}

		m, err := kino.ParseMask("a,-b,c:(d,-e),-z:(x)")
		require.NoError(t, err)
		require.ElementsMatch(t, want, Project(m))
	})

	t.Run("-a:(-b:(-c:(d:(e,-f),-g,y:(z,-w)))) negative with children projection", func(t *testing.T) {
		want := bson.D{
			{Key: "a.b.c.d.e", Value: 1},
			{Key: "a.b.c.y.z", Value: 1},
		}

		m, err := kino.ParseMask("-a:(-b:(-c:(d:(e,-f),-g,y:(z,-w))))")
		m.Mode = kino.Negative // force negative mode - silly little edge case of an unsupported feature
		require.NoError(t, err)
		require.ElementsMatch(t, want, Project(m))
	})

	t.Run("a,-b:(c),d:(e),f:(g:(h)),-i mixed inclusion exclusion projection", func(t *testing.T) {
		want := bson.D{
			{Key: "a", Value: 1},
			{Key: "b.c", Value: 1},
			{Key: "d.e", Value: 1},
			{Key: "f.g.h", Value: 1},
		}

		m, err := kino.ParseMask("a,-b:(c),d:(e),f:(g:(h)),-i")
		require.NoError(t, err)
		require.ElementsMatch(t, want, Project(m))
	})
}
