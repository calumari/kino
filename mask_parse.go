package kino

import (
	"fmt"
	"strings"
	"unicode"
)

type parseState int

const (
	stateParseField parseState = iota + 1
	stateAfterField
	stateDescend
	stateAscend
	stateTraverse
)

func ParseMask(s string) (*Mask, error) {
	idx := 0
	currentState := stateParseField
	root := &Mask{Mode: Positive, Fields: make(map[string]*Node)}
	stack := []*Mask{root}
	var fieldName string
	var op Op
	pendingField := false
	seenInclude := false
	seenExclude := false

	addLeaf := func(name string, op Op) error {
		if name == "" {
			return fmt.Errorf("empty field at index %d", idx)
		}
		cur := stack[len(stack)-1]
		if _, exists := cur.Fields[name]; exists {
			return fmt.Errorf("duplicate field '%s' at index %d", name, idx)
		}
		cur.Fields[name] = &Node{Op: op}
		if op == Positive {
			seenInclude = true
		} else {
			seenExclude = true
		}
		return nil
	}
	startSubtree := func(name string, op Op) error {
		if name == "" {
			return fmt.Errorf("empty field before ':' at index %d", idx)
		}
		cur := stack[len(stack)-1]
		if _, exists := cur.Fields[name]; exists {
			return fmt.Errorf("duplicate field '%s' at index %d", name, idx)
		}
		child := &Mask{Mode: Positive, Fields: make(map[string]*Node)}
		cur.Fields[name] = &Node{Op: op, Children: child}
		if op == Positive {
			seenInclude = true
		} else {
			seenExclude = true
		}
		stack = append(stack, child)
		return nil
	}
	skipSpaces := func() {
		for idx < len(s) && unicode.IsSpace(rune(s[idx])) {
			idx++
		}
	}

	skipSpaces()
	for idx <= len(s) {
		switch currentState {
		case stateParseField:
			if idx >= len(s) {
				if pendingField {
					return nil, fmt.Errorf("internal: unexpected EOF while parsing field")
				}
				idx++
				break
			}
			if s[idx] == '-' {
				op = Negative
				idx++
			} else {
				op = Positive
			}
			skipSpaces()
			start := idx
			for idx < len(s) {
				c := s[idx]
				if c == ':' || c == ',' || c == ')' {
					break
				}
				if !unicode.IsSpace(rune(c)) {
					idx++
					continue
				}
				idx++
			}
			fieldName = strings.TrimSpace(s[start:idx])
			pendingField = true
			currentState = stateAfterField
			continue
		case stateAfterField:
			if !pendingField {
				return nil, fmt.Errorf("parser: stateAfterField without pending field at index %d", idx)
			}
			if idx >= len(s) {
				if err := addLeaf(fieldName, op); err != nil {
					return nil, err
				}
				pendingField = false
				idx++
				break
			}
			c := s[idx]
			switch c {
			case ':':
				currentState = stateDescend
			case ',':
				if err := addLeaf(fieldName, op); err != nil {
					return nil, err
				}
				pendingField = false
				currentState = stateTraverse
			case ')':
				if err := addLeaf(fieldName, op); err != nil {
					return nil, err
				}
				pendingField = false
				currentState = stateAscend
			default:
				if unicode.IsSpace(rune(c)) {
					idx++
					continue
				}
				return nil, fmt.Errorf("unexpected '%c' after field '%s' at index %d", c, fieldName, idx)
			}
			if currentState != stateAfterField {
				idx++
			}
			continue
		case stateDescend:
			skipSpaces()
			if idx >= len(s) || s[idx] != '(' {
				return nil, fmt.Errorf("expected '(' after ':' for field '%s' at index %d", fieldName, idx)
			}
			idx++
			skipSpaces()
			if idx < len(s) && s[idx] == ')' {
				return nil, fmt.Errorf("empty subtree for field '%s' at index %d", fieldName, idx)
			}
			if err := startSubtree(fieldName, op); err != nil {
				return nil, err
			}
			pendingField = false
			currentState = stateParseField
			skipSpaces()
			continue
		case stateTraverse:
			skipSpaces()
			if idx >= len(s) {
				return nil, fmt.Errorf("trailing comma at end of input")
			}
			currentState = stateParseField
			continue
		case stateAscend:
			if len(stack) <= 1 {
				return nil, fmt.Errorf("unmatched ')' at index %d", idx)
			}
			stack = stack[:len(stack)-1]
			pendingField = false
			skipSpaces()
			if idx < len(s) {
				if s[idx] == ',' {
					idx++
					currentState = stateParseField
					skipSpaces()
					continue
				}
				if s[idx] == ')' {
					idx++
					currentState = stateAscend
					continue
				}
				if unicode.IsSpace(rune(s[idx])) {
					continue
				}
				return nil, fmt.Errorf("expected ',' or ')' at index %d", idx)
			}
			idx++
			continue
		default:
			return nil, fmt.Errorf("unknown state %d at index %d", currentState, idx)
		}
	}
	if len(stack) != 1 {
		return nil, fmt.Errorf("unexpected end of string: missing closing ')'")
	}
	// Auto-detect root mode: only exclusions present => blacklist
	if !seenInclude && seenExclude {
		root.Mode = Negative
	}
	return root, nil
}
