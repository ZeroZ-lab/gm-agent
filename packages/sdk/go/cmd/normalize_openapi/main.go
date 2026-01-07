package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	input := flag.String("in", "", "input swagger yaml")
	output := flag.String("out", "", "output swagger yaml")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: normalize_openapi -in <input> -out <output>")
		os.Exit(2)
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "parse yaml: %v\n", err)
		os.Exit(1)
	}

	refs := map[string]string{}
	if defs, ok := doc["definitions"].(map[string]any); ok {
		renamed := map[string]any{}
		for k, v := range defs {
			newKey := strings.ReplaceAll(k, ".", "_")
			if newKey != k {
				refs["#/definitions/"+k] = "#/definitions/" + newKey
			}
			renamed[newKey] = v
		}
		doc["definitions"] = renamed
	}

	if len(refs) > 0 {
		rewriteRefs(doc, refs)
	}

	outData, err := yaml.Marshal(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal yaml: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, outData, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
}

func rewriteRefs(node any, refs map[string]string) {
	switch v := node.(type) {
	case map[string]any:
		for key, value := range v {
			if key == "$ref" {
				if s, ok := value.(string); ok {
					if repl, ok := refs[s]; ok {
						v[key] = repl
					}
				}
				continue
			}
			rewriteRefs(value, refs)
		}
	case []any:
		for i := range v {
			rewriteRefs(v[i], refs)
		}
	}
}
