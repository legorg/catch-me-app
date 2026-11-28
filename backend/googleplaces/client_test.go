package googleplaces

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"catch-me/backend/game"
)

func TestLandmarksSendsRectangleAndFiltersResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		want := map[string]any{
			"textQuery": "tourist attraction",
			"pageSize":  float64(20),
			"locationRestriction": map[string]any{"rectangle": map[string]any{
				"low":  map[string]any{"latitude": 52.4, "longitude": 13.3},
				"high": map[string]any{"latitude": 52.6, "longitude": 13.5},
			}},
		}
		wantJSON, _ := json.Marshal(want)
		gotJSON, _ := json.Marshal(got)
		if r.Method != http.MethodPost || r.URL.Path != "/v1/places:searchText" ||
			r.Header.Get("X-Goog-Api-Key") != "secret" ||
			r.Header.Get("X-Goog-FieldMask") != "places.displayName,places.location" ||
			r.Header.Get("Content-Type") != "application/json" || string(wantJSON) != string(gotJSON) {
			t.Errorf("unexpected request: %s %s %v", r.Method, r.URL.Path, got)
		}
		_, _ = w.Write([]byte(`{"places":[
			{"displayName":{"text":"Inside"},"location":{"latitude":52.5,"longitude":13.4}},
			{"displayName":{"text":"Outside"},"location":{"latitude":53.5,"longitude":13.4}},
			{"displayName":{"text":""},"location":{"latitude":52.5,"longitude":13.4}},
			{"displayName":{"text":"NoLocation"}}]}`))
	}))
	defer server.Close()

	got, err := NewClient(server.URL, "secret", "").Landmarks(context.Background(), game.Bounds{North: 52.6, South: 52.4, East: 13.5, West: 13.3})
	if err != nil || len(got) != 1 || got[0].Title != "Inside" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestLandmarksSendsLanguageCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if got["languageCode"] != "de" {
			t.Errorf("languageCode = %v", got["languageCode"])
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	got, err := NewClient(server.URL, "k", "de").Landmarks(context.Background(), game.Bounds{North: 1, South: 0, East: 1, West: 0})
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestLandmarksStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(429) }))
	defer server.Close()
	if _, err := NewClient(server.URL, "k", "").Landmarks(context.Background(), game.Bounds{North: 1, South: 0, East: 1, West: 0}); err == nil {
		t.Fatal("expected error")
	}
}

func TestLandmarksWithoutAPIKeySendsNoRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request without an API key")
	}))
	defer server.Close()
	if _, err := NewClient(server.URL, "", "").Landmarks(context.Background(), game.Bounds{North: 1, South: 0, East: 1, West: 0}); !errors.Is(err, ErrNoAPIKey) {
		t.Fatalf("err = %v, want ErrNoAPIKey", err)
	}
}
