package kino

import (
	"fmt"
	"sort"
	"strings"
)

type Op int

const (
	Positive Op = iota
	Negative
)

type Node struct {
	Op       Op
	Children *Mask
}

// Mask represents a field projection tree. Fields maps field name -> Node
// (include/exclude + optional subtree). Mode governs root semantics (whitelist
// vs blacklist).
type Mask struct {
	// // Mode controls how a mask is applied when projecting values.
	//   - Include (default) includes only the explicitly listed positive paths.
	//   - Exclude includes everything except explicitly listed negative paths.
	Mode   Op
	Fields map[string]*Node
}

func (m *Mask) String() string {
	if m == nil || len(m.Fields) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m.Fields))
	for k := range m.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(m.Fields))
	for _, name := range keys {
		node := m.Fields[name]
		prefix := ""
		if node.Op == Negative {
			prefix = "-"
		}
		if node.Children != nil && len(node.Children.Fields) > 0 {
			parts = append(parts, fmt.Sprintf("%s%s:(%s)", prefix, name, node.Children))
		} else {
			parts = append(parts, prefix+name)
		}
	}
	return strings.Join(parts, ",")
}

// Overlay returns a new Mask that is the field-wise union of the receiver and
// other. The receiver's existing field Ops always win; only missing fields (or
// missing child subtrees) are taken from other. Resulting nodes are deep copies
// (inputs are never mutated) and the root Mode of every merged node is
// recomputed from its direct children (negative-only => Negative else
// Positive). A nil receiver or nil other is treated as an empty mask.
func (m *Mask) Overlay(other *Mask) *Mask {
	return overlayMaskRecursive(m, other)
}

// overlayMaskRecursive is the implementation behind Mask.Overlay. It performs a
// structural, immutable merge of two masks:
//   - Base fields copied first (deep clone).
//   - Overlay fields added only when absent in base.
//   - For overlapping keys: base Op retained; children merged recursively.
//   - Modes are ignored during merge and derived fresh from each merged node's
//     direct children (deriveMode).
//
// Complexity is proportional to the number of distinct nodes visited. Inputs
// are never mutated; a brand new tree is returned (or a clone when one side is
// nil).
func overlayMaskRecursive(base, overlay *Mask) *Mask {
	if base == nil {
		return cloneMask(overlay)
	}
	if overlay == nil {
		return cloneMask(base)
	}

	res := &Mask{ // start with cloned base fields
		Mode:   base.Mode, // temporary; will be recomputed below
		Fields: make(map[string]*Node, len(base.Fields)+len(overlay.Fields)),
	}
	for k, n := range base.Fields {
		res.Fields[k] = cloneNode(n)
	}

	// merge in overlay fields.
	for k, oNode := range overlay.Fields {
		if existing, ok := res.Fields[k]; ok {
			// preserve existing.Op. merge/attach children recursively.
			if oNode.Children != nil {
				if existing.Children == nil {
					existing.Children = cloneMask(oNode.Children)
				} else {
					existing.Children = overlayMaskRecursive(existing.Children, oNode.Children)
				}
			}
			continue // keep base Op
		}
		res.Fields[k] = cloneNode(oNode)
	}
	res.Mode = deriveMode(res)

	return res
}

// cloneMask returns a deep copy of m (recursively cloning children). Nil safe.
func cloneMask(m *Mask) *Mask {
	if m == nil {
		return nil
	}
	cp := &Mask{
		Mode:   m.Mode,
		Fields: make(map[string]*Node, len(m.Fields)),
	}
	for k, n := range m.Fields {
		cp.Fields[k] = cloneNode(n)
	}
	return cp
}

// cloneNode returns a deep copy of n (recursively cloning its Children). Nil
// safe.
func cloneNode(n *Node) *Node {
	if n == nil {
		return nil
	}
	cp := &Node{Op: n.Op}
	if n.Children != nil {
		cp.Children = cloneMask(n.Children)
	}
	return cp
}

// deriveMode determines a mask's Mode from its immediate children only. It is
// Negative iff there is at least one negative child and zero positive children;
// otherwise Positive. Nested descendants do not influence the mode of an
// ancestor. Nil / empty => Positive.
func deriveMode(m *Mask) Op {
	if m == nil || len(m.Fields) == 0 {
		return Positive
	}
	hasPos, hasNeg := false, false
	for _, n := range m.Fields {
		switch n.Op {
		case Positive:
			hasPos = true
		case Negative:
			hasNeg = true
		}
		if hasPos && hasNeg { // can't become negative-only anymore
			return Positive
		}
	}
	if !hasPos && hasNeg {
		return Negative
	}
	return Positive
}
