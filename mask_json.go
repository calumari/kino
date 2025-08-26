package kino

import (
	"encoding/json"
	"fmt"
)

func (m *Mask) MarshalJSON() ([]byte, error) {
	if m == nil || len(m.Fields) == 0 {
		return []byte("{}"), nil
	}
	var walk func(mm *Mask) map[string]any
	walk = func(mm *Mask) map[string]any {
		x := make(map[string]any, len(mm.Fields))
		for k, n := range mm.Fields {
			if n.Children != nil && len(n.Children.Fields) > 0 {
				x[k] = walk(n.Children)
			} else {
				x[k] = n.Op == Positive
			}
		}
		return x
	}
	x := walk(m)
	return json.Marshal(x)
}

// Legacy encoding/json (v1) fallback: still supports flat map[string]bool form.
func (m *Mask) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*m = Mask{}
		return nil
	}
	// Try nested object first using standard decode.
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		return err
	}
	var build func(mm map[string]any) (*Mask, error)
	build = func(mm map[string]any) (*Mask, error) {
		res := &Mask{Mode: Positive, Fields: make(map[string]*Node, len(mm))}
		for k, v := range mm {
			switch vv := v.(type) {
			case map[string]any:
				child, err := build(vv)
				if err != nil {
					return nil, err
				}
				res.Fields[k] = &Node{Op: Positive, Children: child}
			case bool:
				op := Negative
				if vv {
					op = Positive
				}
				res.Fields[k] = &Node{Op: op}
			case float64:
				op := Positive
				if vv < 0 {
					op = Negative
				}
				res.Fields[k] = &Node{Op: op}
			default:
				return nil, fmt.Errorf("unsupported value type %T for key %q", v, k)
			}
		}
		// Determine mode for this mask: negative-only => Negative.
		hasPos, hasNeg := false, false
		for _, n := range res.Fields {
			if n.Op == Positive {
				hasPos = true
			} else if n.Op == Negative {
				hasNeg = true
			}
		}
		if !hasPos && hasNeg {
			res.Mode = Negative
		} else {
			res.Mode = Positive
		}
		return res, nil
	}
	built, err := build(generic)
	if err != nil {
		return err
	}
	if built != nil {
		*m = *built
	}
	return nil
}
