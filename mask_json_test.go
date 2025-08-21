package kino_test

import (
	"bytes"
	"testing"

	"github.com/calumari/kino"
	"github.com/go-json-experiment/json"
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

		var buf bytes.Buffer
		err := json.MarshalWrite(&buf, m)
		require.NoError(t, err)
		require.JSONEq(t, `{"a":true,"b":false,"c":{"d":true,"e":false}}`, buf.String())

		var m2 kino.Mask
		err = json.UnmarshalRead(bytes.NewReader(buf.Bytes()), &m2)
		require.NoError(t, err)

		var out bytes.Buffer
		err = json.MarshalWrite(&out, m2)
		require.NoError(t, err)
		require.JSONEq(t, buf.String(), out.String())
	})
}
