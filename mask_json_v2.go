package kino

import (
	"bytes"
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// MaskUnmarshalers returns a json.Unmarshalers helper that can decode a JSON
// object into a Mask value. It recognises nested objects and leaf
// booleans/numbers (negative meaning exclusion).
func MaskUnmarshalers() *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *Mask) error {
		if dec.PeekKind() != '{' {
			return json.SkipFunc
		}
		if _, err := dec.ReadToken(); err != nil { // consume opening '{'
			return fmt.Errorf("read opening '{': %w", err)
		}
		if v.Fields == nil {
			v.Fields = make(map[string]*Node)
		}
		mask := v

		// Track pos/neg for this object (nested objects recurse via another
		// invocation, so a single frame is sufficient here unlike the parser's
		// multi-level stack during expression parsing).
		type frame struct {
			hasPos, hasNeg bool
		}
		f := &frame{}
		update := func(op Op) {
			if op == Positive {
				f.hasPos = true
			} else {
				f.hasNeg = true
			}
		}
		for dec.PeekKind() != '}' {
			// read key
			var key string
			if err := json.UnmarshalDecode(dec, &key); err != nil {
				return fmt.Errorf("read key: %w", err)
			}
			if _, exists := mask.Fields[key]; exists {
				return fmt.Errorf("duplicate field %q", key)
			}

			switch dec.PeekKind() {
			case '{':
				var child Mask
				if err := json.UnmarshalDecode(dec, &child); err != nil {
					return fmt.Errorf("decode child %q: %w", key, err)
				}
				mask.Fields[key] = &Node{Op: Positive, Children: &child}
				update(Positive)
			default:
				var raw any
				if err := json.UnmarshalDecode(dec, &raw); err != nil {
					return fmt.Errorf("read value for %q: %w", key, err)
				}
				op := Positive
				switch v := raw.(type) {
				case bool:
					if !v {
						op = Negative
					}
				case float64:
					if v < 0 {
						op = Negative
					}
				case int64:
					if v < 0 {
						op = Negative
					}
				case uint64:
					// always positive
				default:
					return fmt.Errorf("unexpected value type %T for key %q", raw, key)
				}
				mask.Fields[key] = &Node{Op: op}
				update(op)
			}
		}
		if _, err := dec.ReadToken(); err != nil { // consume closing '}'
			return fmt.Errorf("read closing '}': %w", err)
		}
		// finalize mode for this mask (negative-only => Negative)
		if !f.hasPos && f.hasNeg {
			mask.Mode = Negative
		} else {
			mask.Mode = Positive
		}
		return nil
	})
}

// WithMask returns a json.Marshalers helper that, when supplied to
// json.Marshal, projects arbitrary input values according to mask m (only
// positive paths are emitted; negative or absent paths are omitted).
func MarshalWithMask(m *Mask) *json.Marshalers {
	return json.MarshalToFunc(func(enc *jsontext.Encoder, v any) error {
		var buf bytes.Buffer
		if err := json.MarshalWrite(&buf, v); err != nil {
			return fmt.Errorf("marshal mask source: %w", err)
		}
		dec := jsontext.NewDecoder(&buf)

		// copyRaw copies the next value from dec to enc verbatim.
		var copyRaw func() error
		copyRaw = func() error {
			switch dec.PeekKind() {
			case '{':
				if _, err := dec.ReadToken(); err != nil {
					return fmt.Errorf("read '{': %w", err)
				}
				if err := enc.WriteToken(jsontext.BeginObject); err != nil {
					return fmt.Errorf("write '{': %w", err)
				}
				for dec.PeekKind() != '}' {
					var key string
					if err := json.UnmarshalDecode(dec, &key); err != nil {
						return fmt.Errorf("read key (raw copy): %w", err)
					}
					if err := enc.WriteToken(jsontext.String(key)); err != nil {
						return fmt.Errorf("write key (raw copy): %w", err)
					}
					if err := copyRaw(); err != nil {
						return err
					}
				}
				if _, err := dec.ReadToken(); err != nil {
					return fmt.Errorf("read '}': %w", err)
				}
				if err := enc.WriteToken(jsontext.EndObject); err != nil {
					return fmt.Errorf("write '}': %w", err)
				}
			case '[':
				if _, err := dec.ReadToken(); err != nil {
					return fmt.Errorf("read '[': %w", err)
				}
				if err := enc.WriteToken(jsontext.BeginArray); err != nil {
					return fmt.Errorf("write '[': %w", err)
				}
				for dec.PeekKind() != ']' {
					if err := copyRaw(); err != nil {
						return err
					}
				}
				if _, err := dec.ReadToken(); err != nil {
					return fmt.Errorf("read ']': %w", err)
				}
				if err := enc.WriteToken(jsontext.EndArray); err != nil {
					return fmt.Errorf("write ']': %w", err)
				}
			default:
				tok, err := dec.ReadToken()
				if err != nil {
					return fmt.Errorf("read scalar: %w", err)
				}
				if err := enc.WriteToken(tok); err != nil {
					return fmt.Errorf("write scalar: %w", err)
				}
			}
			return nil
		}

		// copyMasked copies the next value applying the provided mask.
		var copyMasked func(mask *Mask, ancestorExcluded bool) error
		copyMasked = func(mask *Mask, ancestorExcluded bool) error {
			// No mask means copy everything.
			if mask == nil {
				return copyRaw()
			}
			// Choose algorithm based on root mode and context.
			switch dec.PeekKind() {
			case '{':
				if _, err := dec.ReadToken(); err != nil {
					return fmt.Errorf("read '{': %w", err)
				}
				if err := enc.WriteToken(jsontext.BeginObject); err != nil {
					return fmt.Errorf("write '{': %w", err)
				}
				for dec.PeekKind() != '}' {
					var key string
					if err := json.UnmarshalDecode(dec, &key); err != nil {
						return fmt.Errorf("read key: %w", err)
					}
					node, ok := mask.Fields[key]
					if mask.Mode == Positive || ancestorExcluded {
						// Whitelist semantics (or inside an excluded-override
						// subtree): Only explicitly included paths are emitted.
						//
						// Override support for pattern: -parent:(child,...) If
						// a negative node has children we treat it as an
						// exclusion of the whole subtree with selective
						// re-includes of its positive descendants, even when
						// the overall root mode was detected as Positive (this
						// can happen today because root mode auto-detection
						// counts nested positives). This enables the documented
						// expression `-z:(x)` to yield `{"z":{"x":..}}`.
						if !ok || node.Op == Negative && (node.Children == nil || len(node.Children.Fields) == 0) {
							// Simple negative (no override children) or absent
							// field: skip.
							if err := dec.SkipValue(); err != nil {
								return fmt.Errorf("skip masked value %q: %w", key, err)
							}
							continue
						}
						if ok && node.Op == Negative && node.Children != nil && len(node.Children.Fields) > 0 {
							// Negative-with-children override: emit key and
							// descend with whitelist semantics limited to the
							// provided children.
							if err := enc.WriteToken(jsontext.String(key)); err != nil {
								return fmt.Errorf("write key %q: %w", key, err)
							}
							if err := copyMasked(&Mask{Mode: Positive, Fields: node.Children.Fields}, true); err != nil {
								return err
							}
							continue
						}
						// Standard positive include path.
						if err := enc.WriteToken(jsontext.String(key)); err != nil {
							return fmt.Errorf("write key %q: %w", key, err)
						}
						if node.Children != nil && len(node.Children.Fields) > 0 {
							if err := copyMasked(node.Children, false); err != nil {
								return err
							}
						} else if err := copyRaw(); err != nil {
							return err
						}
					} else { // ModeExclude outside override
						// Blacklist semantics: skip only explicit excludes;
						// includes can narrow subtrees.
						if ok && node.Op == Negative {
							if node.Children == nil || len(node.Children.Fields) == 0 {
								if err := dec.SkipValue(); err != nil {
									return fmt.Errorf("skip excluded %q: %w", key, err)
								}
								continue
							}
							// Excluded subtree with potential re-includes: we
							// need to read object and only output included
							// children. Start object in output regardless
							// (unless everything filtered -> produce empty
							// object). Write key then filter children with
							// ancestorExcluded=true.
							if err := enc.WriteToken(jsontext.String(key)); err != nil {
								return err
							}
							// Descend marking ancestorExcluded=true so only
							// explicit includes survive.
							if err := copyMasked(&Mask{Mode: Positive, Fields: node.Children.Fields}, true); err != nil {
								return err
							}
						} else {
							// Included by default or explicitly included.
							if err := enc.WriteToken(jsontext.String(key)); err != nil {
								return err
							}
							if ok && node.Children != nil && len(node.Children.Fields) > 0 {
								if err := copyMasked(node.Children, false); err != nil {
									return err
								}
							} else if err := copyRaw(); err != nil {
								return err
							}
						}
					}
				}
				if _, err := dec.ReadToken(); err != nil {
					return fmt.Errorf("read '}': %w", err)
				}
				if err := enc.WriteToken(jsontext.EndObject); err != nil {
					return fmt.Errorf("write '}': %w", err)
				}
			case '[':
				// Apply same mask to each element.
				if _, err := dec.ReadToken(); err != nil {
					return fmt.Errorf("read '[': %w", err)
				}
				if err := enc.WriteToken(jsontext.BeginArray); err != nil {
					return fmt.Errorf("write '[': %w", err)
				}
				for dec.PeekKind() != ']' {
					if err := copyMasked(mask, ancestorExcluded); err != nil {
						return err
					}
				}
				if _, err := dec.ReadToken(); err != nil {
					return fmt.Errorf("read ']': %w", err)
				}
				if err := enc.WriteToken(jsontext.EndArray); err != nil {
					return fmt.Errorf("write ']': %w", err)
				}
			default:
				// Scalar at a masked location: copy if we reached here (meaning
				// parent allowed it).
				return copyRaw()
			}
			return nil
		}

		return copyMasked(m, false)
	})
}
