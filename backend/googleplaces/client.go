// Package googleplaces finds landmarks through the Google Places API (New) Text Search.
package googleplaces

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"catch-me/backend/game"
)

// DefaultBaseURL is the production Places API (New) endpoint.
const DefaultBaseURL = "https://places.googleapis.com"

// ErrNoAPIKey means the client has no key, so it cannot look up landmarks.
var ErrNoAPIKey = errors.New("google places: no API key configured")

const (
	textQuery = "tourist attraction"
	pageSize  = 20
	fieldMask = "places.displayName,places.location"
)

type Client struct {
	baseURL      string
	apiKey       string
	languageCode string
	http         *http.Client
}

// CallTimeout bounds one Places request, so a hanging call leaves time to try
// another cell within the edge budget (service.edgeLookupTimeout).
const CallTimeout = 3 * time.Second

// NewClient talks to the Places API at baseURL (use DefaultBaseURL in production).
// languageCode is optional. Without apiKey every lookup fails with ErrNoAPIKey.
func NewClient(baseURL, apiKey, languageCode string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, languageCode: languageCode, http: &http.Client{Timeout: CallTimeout}}
}

type latLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type searchRequest struct {
	TextQuery           string `json:"textQuery"`
	PageSize            int    `json:"pageSize"`
	LanguageCode        string `json:"languageCode,omitempty"`
	LocationRestriction struct {
		Rectangle struct {
			Low  latLng `json:"low"`
			High latLng `json:"high"`
		} `json:"rectangle"`
	} `json:"locationRestriction"`
}

// Landmarks returns named places located inside bounds, which is meant to be one grid cell.
// Results must be shown on a Google Map or with Google Maps attribution.
func (c *Client) Landmarks(ctx context.Context, bounds game.Bounds) ([]game.Landmark, error) {
	if c.apiKey == "" {
		return nil, ErrNoAPIKey
	}
	payload := searchRequest{TextQuery: textQuery, PageSize: pageSize, LanguageCode: c.languageCode}
	payload.LocationRestriction.Rectangle.Low = latLng{bounds.South, bounds.West}
	payload.LocationRestriction.Rectangle.High = latLng{bounds.North, bounds.East}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/places:searchText", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Goog-Api-Key", c.apiKey)
	request.Header.Set("X-Goog-FieldMask", fieldMask)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("google places: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google places: status %d", response.StatusCode)
	}
	var body struct {
		Places []struct {
			DisplayName struct {
				Text string `json:"text"`
			} `json:"displayName"`
			Location *latLng `json:"location"`
		} `json:"places"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode google places response: %w", err)
	}
	var landmarks []game.Landmark
	for _, place := range body.Places {
		if place.Location == nil || place.DisplayName.Text == "" {
			continue
		}
		coordinates, err := game.ParseCoordinates(place.Location.Latitude, place.Location.Longitude)
		if err != nil || !bounds.Contains(coordinates) {
			continue
		}
		landmarks = append(landmarks, game.Landmark{Title: place.DisplayName.Text, Coordinates: coordinates})
	}
	return landmarks, nil
}
