package kino_test

import (
	"encoding/json"
	"testing"

	"github.com/calumari/kino"
	"github.com/stretchr/testify/require"
)

func TestMaskJSON(t *testing.T) {
	t.Run("legacy marshal/unmarshal round trip success", func(t *testing.T) {
		m := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Positive},
			"b": {Op: kino.Negative},
			"c": nodePos(maskPositive(map[string]*kino.Node{
				"d": {Op: kino.Positive},
				"e": {Op: kino.Negative},
			})),
		})
		data, err := json.Marshal(m)
		require.NoError(t, err)
		require.JSONEq(t, `{"a":true,"b":false,"c":{"d":true,"e":false}}`, string(data))

		var m2 kino.Mask
		err = json.Unmarshal(data, &m2)
		require.NoError(t, err)
		data2, err := json.Marshal(&m2)
		require.NoError(t, err)
		require.JSONEq(t, string(data), string(data2))
	})

	t.Run("legacy unmarshal nested negative-only subtree sets negative mode", func(t *testing.T) {
		data := []byte(`{"meta":{"secret":false}}`)
		var m kino.Mask
		require.NoError(t, json.Unmarshal(data, &m))
		// Root has a positive child (object) so stays Positive.
		require.Equal(t, kino.Positive, m.Mode)
		child := m.Fields["meta"].Children
		require.NotNil(t, child)
		// Child contains only a negative => negative mode.
		require.Equal(t, kino.Negative, child.Mode)
	})
}
