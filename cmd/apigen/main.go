package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flowc-labs/flowc/internal/flowc/apigen"
	"gopkg.in/yaml.v3"
)

func main() {
	doc := apigen.Generate()

	// Use kin-openapi's MarshalJSON for correct serialization of all
	// custom types (ordered maps, extensions, etc.), then convert to YAML.
	jsonData, err := doc.MarshalJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}

	var generic interface{}
	if err := json.Unmarshal(jsonData, &generic); err != nil {
		fmt.Fprintf(os.Stderr, "failed to unmarshal JSON: %v\n", err)
		os.Exit(1)
	}

	yamlData, err := yaml.Marshal(generic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal YAML: %v\n", err)
		os.Exit(1)
	}

	outPath := filepath.Join("api", "openapi.yaml")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, yamlData, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", outPath)
}
