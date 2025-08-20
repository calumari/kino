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

// Mask represents a field projection tree.
// Fields maps field name -> Node (include/exclude + optional subtree).
// Mode governs root semantics (whitelist vs blacklist).
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
