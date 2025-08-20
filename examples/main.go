package main

import (
	"fmt"
	"log"

	"github.com/calumari/kino"
	"github.com/go-json-experiment/json"
)

func main() {
	// Example mask string syntax:
	//  a         include field a
	//  -b        exclude field b
	//  c:(d,-e)  include c.d, exclude c.e
	//  f:(g,h)   include f.g and f.h
	//  i         include i
	//  -z:(-y,x) exclude z but keep z.x (and still exclude z.y)
	maskExpr := "a,-b,c:(d,-e),f:(g,h),i,-z:(-y,x)"
	mask, err := kino.ParseMask(maskExpr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Parsed mask expression:", maskExpr)
	fmt.Println("Mask.String():         ", mask.String())

	// Show JSON form of the mask (structure with booleans).
	maskJSONBytes, err := json.Marshal(mask)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Mask as JSON:          ", string(maskJSONBytes))

	// Reconstruct mask from JSON using experimental unmarshalers.
	var maskFromJSON kino.Mask
	if err := json.Unmarshal(maskJSONBytes, &maskFromJSON, json.WithUnmarshalers(kino.MaskUnmarshalers())); err != nil {
		log.Fatal(err)
	}
	// Structural equality: compare JSON normalized forms.
	origJSON, _ := json.Marshal(mask)
	rtJSON, _ := json.Marshal(maskFromJSON)
	fmt.Println("Round-trip equal?      ", string(origJSON) == string(rtJSON))

	// Sample data to project with mask.
	type Inner struct {
		D string `json:"d"`
		E string `json:"e"`
	}
	type Sample struct {
		A string `json:"a"`
		B string `json:"b"`
		C Inner  `json:"c"`
		I string `json:"i"`
		Z struct {
			X string `json:"x"`
			Y string `json:"y"`
		} `json:"z"`
	}

	s := Sample{
		A: "show A",
		B: "hide B",
		C: Inner{D: "show C.D", E: "hide C.E"},
		I: "show I",
		Z: struct {
			X string `json:"x"`
			Y string `json:"y"`
		}{X: "show Z.X", Y: "hide Z.Y"},
	}

	// Mask also applies element-wise to arrays/slices.
	slice := []Sample{s, s}

	// Project single object.
	projectedObj, err := json.Marshal(s, json.WithMarshalers(kino.MarshalWithMask(mask)))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Masked object:         ", string(projectedObj))

	// Project slice.
	projectedSlice, err := json.Marshal(slice, json.WithMarshalers(kino.MarshalWithMask(mask)))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Masked slice:          ", string(projectedSlice))

	// Exclusion mode example: only hide keys explicitly marked with '-' in the
	// mask.
	excludedObj, err := json.Marshal(s, json.WithMarshalers(kino.MarshalWithMask(mask)))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Exclusion mode object: ", string(excludedObj))

	// Quick diff-like view: list removed top-level fields.
	rawObj, _ := json.Marshal(s)
	fmt.Println("Original keys present: ", topLevelKeys(string(rawObj)))
	fmt.Println("Masked keys present:   ", topLevelKeys(string(projectedObj)))
}

// topLevelKeys extracts JSON top-level object keys (best-effort, assumes no
// escapes).
func topLevelKeys(s string) []string {
	if len(s) == 0 || s[0] != '{' {
		return nil
	}
	var keys []string
	inStr := false
	esc := false
	depth := -1
	keyStart := -1
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
			if depth == 0 {
				keyStart = i + 1
			}
		case ':':
			if depth == 0 && keyStart > 0 {
				// look back to preceding quote
				j := i - 1
				for j >= 0 && s[j] != '"' {
					j--
				}
				if j > keyStart-1 {
					keys = append(keys, s[keyStart:j])
				}
			}
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	return keys
}
