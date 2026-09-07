package linters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

type openAPISpec struct {
	OpenAPI string `yaml:"openapi"`
	Info    struct {
		Title   string `yaml:"title"`
		Version string `yaml:"version"`
	} `yaml:"info"`
	Servers []struct {
		URL         string `yaml:"url"`
		Description string `yaml:"description"`
	} `yaml:"servers"`
	Paths map[string]map[string]interface{} `yaml:"paths"`
}

func TestOpenAPISpecsLint(t *testing.T) {
	files, err := filepath.Glob("../../api/*.yaml")
	if err != nil || len(files) == 0 {
		t.Fatalf("failed to find api specs: %v", err)
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("failed to read spec file: %v", err)
			}

			var spec openAPISpec
			if err := yaml.Unmarshal(data, &spec); err != nil {
				t.Fatalf("failed to parse YAML: %v", err)
			}

			// 1. Enforce OpenAPI version
			if spec.OpenAPI == "" {
				t.Errorf("missing openapi version declaration")
			}

			// 2. Enforce servers block containing /v1
			if len(spec.Servers) == 0 {
				t.Errorf("missing 'servers:' section in OpenAPI spec")
			} else {
				hasV1 := false
				for _, srv := range spec.Servers {
					if strings.Contains(srv.URL, "/v1") {
						hasV1 = true
						break
					}
				}
				if !hasV1 {
					t.Errorf("expected at least one server URL to contain '/v1' path prefix, got: %+v", spec.Servers)
				}
			}

			// 3. Enforce operationId on every path operation
			for pathName, operations := range spec.Paths {
				for method, opData := range operations {
					// Skip parameters or non-http method keys
					if method == "parameters" {
						continue
					}
					opMap, ok := opData.(map[interface{}]interface{})
					if !ok {
						continue
					}
					if _, hasOpID := opMap["operationId"]; !hasOpID {
						t.Errorf("path '%s' method '%s' is missing 'operationId'", pathName, method)
					}
				}
			}
		})
	}
}
