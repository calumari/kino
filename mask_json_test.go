package kino_test

import (
	jsonv1 "encoding/json"
	"testing"

	"github.com/calumari/kino"
	"github.com/stretchr/testify/require"
)

func TestMaskJSON(t *testing.T) {
	t.Run("legacy marshal/unmarshal round trip", func(t *testing.T) {
		m := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Positive},
			"b": {Op: kino.Negative},
			"c": nodePos(maskPositive(map[string]*kino.Node{
				"d": {Op: kino.Positive},
				"e": {Op: kino.Negative},
			})),
		})
		data, err := jsonv1.Marshal(m)
		require.NoError(t, err)
		require.JSONEq(t, `{"a":true,"b":false,"c":{"d":true,"e":false}}`, string(data))

		var m2 kino.Mask
		require.NoError(t, jsonv1.Unmarshal(data, &m2))
		data2, err := jsonv1.Marshal(&m2)
		require.NoError(t, err)
		require.JSONEq(t, string(data), string(data2))
	})
}
