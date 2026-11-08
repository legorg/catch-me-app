package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDemoGreeting(t *testing.T) {
	for _, test := range []struct {
		name       string
		query      string
		playerName string
		status     int
	}{
		{"default name", "", "Runner", http.StatusOK},
		{"custom name", "?name=Alex", "Alex", http.StatusOK},
		{"blank name", "?name=", "Runner", http.StatusOK},
		{"trim whitespace", "?name=+Alex+", "Alex", http.StatusOK},
		{"encoded name", "?name=" + url.QueryEscape("Runner & Hunter"), "Runner & Hunter", http.StatusOK},
		{"unicode name", "?name=" + url.QueryEscape(strings.Repeat("\u00e4", 60)), strings.Repeat("\u00e4", 60), http.StatusOK},
		{"name too long", "?name=" + strings.Repeat("x", 61), "", http.StatusBadRequest},
		{"invalid UTF-8", "?name=%FF", "", http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			NewHandler("").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/demo"+test.query, nil))
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d", recorder.Code, test.status)
			}
			var body map[string]string
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if test.status != http.StatusOK {
				if len(body) != 1 || body["error"] == "" {
					t.Fatalf("unexpected error body: %v", body)
				}
				return
			}
			want := map[string]string{
				"message":    fmt.Sprintf("Hello, %s! Your Expo app is connected to Go.", test.playerName),
				"playerName": test.playerName,
				"service":    "catch-me-backend",
			}
			if !reflect.DeepEqual(body, want) {
				t.Fatalf("body = %v, want %v", body, want)
			}
		})
	}
}

// This smoke test compares default responses to examples in the shared contract.
// It is not a general-purpose OpenAPI schema validator.
func TestOpenAPIExamples(t *testing.T) {
	data, err := os.ReadFile("../../api/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Swagger string
		Paths   map[string]struct {
			Get struct {
				Responses map[string]struct {
					Schema struct {
						Ref string `json:"$ref"`
					}
				}
			}
		}
		Definitions map[string]struct {
			Required   []string
			Properties map[string]struct {
				Type    string
				Example string
			}
		}
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Swagger != "2.0" {
		t.Fatalf("expected generated Swagger 2.0 document, got %q", spec.Swagger)
	}
	for _, path := range []string{"/health", "/api/demo"} {
		t.Run(path, func(t *testing.T) {
			operation, ok := spec.Paths[path]
			if !ok {
				t.Fatalf("contract is missing %s", path)
			}
			ref := operation.Get.Responses["200"].Schema.Ref
			schema, ok := spec.Definitions[strings.TrimPrefix(ref, "#/definitions/")]
			if !ok || len(schema.Properties) == 0 {
				t.Fatal("contract needs a 200 response schema")
			}
			want := make(map[string]string)
			for name, property := range schema.Properties {
				if property.Type != "string" || property.Example == "" {
					t.Fatalf("expected string property with a Go example tag: %s", name)
				}
				want[name] = property.Example
			}
			for _, name := range schema.Required {
				if _, ok := want[name]; !ok {
					t.Fatalf("required field %s is missing from the response examples", name)
				}
			}
			recorder := httptest.NewRecorder()
			NewHandler("").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", recorder.Code)
			}
			var body map[string]string
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, want) {
				t.Fatalf("body = %v, contract example = %v", body, want)
			}
		})
	}
}
