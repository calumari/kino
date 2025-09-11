package mongo

// Package mongo exposes helpers to translate kino masks into MongoDB projection
// documents (bson.D)

import (
	"strings"

	"github.com/calumari/kino"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Project converts a kino.Mask into a bson.D projection. Strategy:
//   - Positive root: inclusion list of leaf positive paths (overrides honored).
//   - Negative root with only simple top-level negative leaves: exclusion doc.
//   - Any negative-with-children override forces inclusion expansion.
//
// Limitations: Mixed inclusion/exclusion at top-level (invalid in Mongo) are
// resolved via inclusion expansion.
func Project(m *kino.Mask) bson.D {
	if m == nil || len(m.Fields) == 0 {
		return bson.D{}
	}

	var needsInclusion func(mm *kino.Mask) bool
	needsInclusion = func(mm *kino.Mask) bool {
		for _, n := range mm.Fields {
			if n.Op == kino.Positive {
				return true
			}
			if n.Op == kino.Negative && n.Children != nil && len(n.Children.Fields) > 0 {
				if needsInclusion(n.Children) {
					return true
				}
			}
		}
		return false
	}
	if !needsInclusion(m) {
		out := make(bson.D, 0, len(m.Fields))
		for name, node := range m.Fields {
			if node.Op == kino.Negative && (node.Children == nil || len(node.Children.Fields) == 0) {
				out = append(out, bson.E{Key: name, Value: 0})
			}
		}
		return out
	}

	inc := make(map[string]struct{})
	stack := make([]string, 0, 8)
	var walk func(mm *kino.Mask)
	walk = func(mm *kino.Mask) {
		for name, node := range mm.Fields {
			stack = append(stack, name)
			if node.Op == kino.Negative {
				if node.Children != nil && len(node.Children.Fields) > 0 {
					walk(node.Children)
				}
				stack = stack[:len(stack)-1]
				continue
			}
			if node.Children != nil && len(node.Children.Fields) > 0 {
				walk(node.Children)
				stack = stack[:len(stack)-1]
				continue
			}
			path := strings.Join(stack, ".")
			inc[path] = struct{}{}
			stack = stack[:len(stack)-1]
		}
	}
	walk(m)
	out := make(bson.D, 0, len(inc))
	for k := range inc {
		out = append(out, bson.E{Key: k, Value: 1})
	}
	return out
}
