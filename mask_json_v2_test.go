package kino_test

import (
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/require"

	"github.com/calumari/kino"
)

type sample struct {
	A string `json:"a"`
	B string `json:"b"`
	C struct {
		D int `json:"d"`
		E int `json:"e"`
	} `json:"c"`
	Z struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"z"`
}

func buildSample() sample {
	var s sample
	s.A = "va"
	s.B = "vb"
	s.C.D = 1
	s.C.E = 2
	s.Z.X = 10
	s.Z.Y = 20
	return s
}

func maskPositive(fields map[string]*kino.Node) *kino.Mask {
	return &kino.Mask{Mode: kino.Positive, Fields: fields}
}
func maskNegative(fields map[string]*kino.Node) *kino.Mask {
	return &kino.Mask{Mode: kino.Negative, Fields: fields}
}
func nodePos(children *kino.Mask) *kino.Node {
	return &kino.Node{Op: kino.Positive, Children: children}
}
func nodeNeg(children *kino.Mask) *kino.Node {
	return &kino.Node{Op: kino.Negative, Children: children}
}

func TestMarshalWithMask(t *testing.T) {
	t.Run("nil mask is skip", func(t *testing.T) {
		out, err := json.Marshal(buildSample(), json.WithMarshalers(kino.MarshalWithMask(nil)))
		require.NoError(t, err)
		require.Equal(t, `{"a":"va","b":"vb","c":{"d":1,"e":2},"z":{"x":10,"y":20}}`, string(out))
	})

	t.Run("positive root simple includes projected", func(t *testing.T) {
		m := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Positive},
			"c": nodePos(maskPositive(map[string]*kino.Node{"d": {Op: kino.Positive}})),
		})
		out, err := json.Marshal(buildSample(), json.WithMarshalers(kino.MarshalWithMask(m)))
		require.NoError(t, err)
		require.JSONEq(t, `{"a":"va","c":{"d":1}}`, string(out))
	})

	t.Run("-b,-c:(-e) negative root excludes projected", func(t *testing.T) {
		m := maskNegative(map[string]*kino.Node{
			"b": {Op: kino.Negative},
			"c": nodeNeg(maskPositive(map[string]*kino.Node{"e": {Op: kino.Negative}})),
		})
		out, err := json.Marshal(buildSample(), json.WithMarshalers(kino.MarshalWithMask(m)))
		require.NoError(t, err)
		require.JSONEq(t, `{"a":"va","c":{},"z":{"x":10,"y":20}}`, string(out))
	})

	t.Run("-c:(-e) negative root excludes single projected", func(t *testing.T) {
		m := maskNegative(map[string]*kino.Node{
			"c": nodeNeg(maskPositive(map[string]*kino.Node{"e": {Op: kino.Negative}})),
		})
		out, err := json.Marshal(buildSample(), json.WithMarshalers(kino.MarshalWithMask(m)))
		require.NoError(t, err)
		// All other fields present, c filtered to empty object.
		require.JSONEq(t, `{"a":"va","b":"vb","c":{},"z":{"x":10,"y":20}}`, string(out))
	})

	t.Run("-z:(x) negative override reinclude projected", func(t *testing.T) {
		m := maskPositive(map[string]*kino.Node{ // root positive because inner positive path exists
			"z": nodeNeg(maskPositive(map[string]*kino.Node{"x": {Op: kino.Positive}})),
		})
		out, err := json.Marshal(buildSample(), json.WithMarshalers(kino.MarshalWithMask(m)))
		require.NoError(t, err)
		require.JSONEq(t, `{"z":{"x":10}}`, string(out))
	})

	t.Run("a,-b,c:(d,-e),-z:(x) complex mixed projected", func(t *testing.T) {
		m := maskPositive(map[string]*kino.Node{
			"a": {Op: kino.Positive},
			"b": {Op: kino.Negative},
			"c": nodePos(maskPositive(map[string]*kino.Node{
				"d": {Op: kino.Positive},
				"e": {Op: kino.Negative},
			})),
			"z": nodeNeg(maskPositive(map[string]*kino.Node{
				"x": {Op: kino.Positive},
			})),
		})
		out, err := json.Marshal(buildSample(), json.WithMarshalers(kino.MarshalWithMask(m)))
		require.NoError(t, err)
		require.JSONEq(t, `{"a":"va","c":{"d":1},"z":{"x":10}}`, string(out))
	})

	t.Run("-z:(x) arrays inherit mask projected", func(t *testing.T) {
		m := maskPositive(map[string]*kino.Node{
			"z": nodeNeg(maskPositive(map[string]*kino.Node{"x": {Op: kino.Positive}})),
		})
		arr := []sample{buildSample(), buildSample()}
		out, err := json.Marshal(arr, json.WithMarshalers(kino.MarshalWithMask(m)))
		require.NoError(t, err)
		require.JSONEq(t, `[{"z":{"x":10}},{"z":{"x":10}}]`, string(out))
	})

	t.Run("unmarshalers nested negative-only subtree sets negative mode", func(t *testing.T) {
		var m kino.Mask
		data := []byte(`{"meta":{"secret":false}}`)
		require.NoError(t, json.Unmarshal(data, &m, json.WithUnmarshalers(kino.MaskUnmarshalers())))
		require.Equal(t, kino.Positive, m.Mode)
		child := m.Fields["meta"].Children
		require.NotNil(t, child)
		require.Equal(t, kino.Negative, child.Mode)
	})
}
