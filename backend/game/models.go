// Package game contains gameplay configuration without HTTP or database dependencies.
//
// Values are parsed, not validated: Parse/New functions are the only way to
// obtain them from raw input, so code receiving these types can trust them.
package game

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"time"
)

// ValidationError reports invalid input (HTTP 400).
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// RuleError reports an action the game rules do not allow right now (HTTP 409).
type RuleError struct{ Message string }

func (e *RuleError) Error() string { return e.Message }

var (
	ErrNotFound     = errors.New("game not found")
	ErrUnauthorized = errors.New("missing or invalid player token")
	// ErrJoinCodeTaken is returned by persistence when generated codes collide.
	// ErrLandmarksUnavailable means the landmark lookup failed, ErrNoLandmark
	// that it succeeded but found nothing on an edge of the playground.
	ErrLandmarksUnavailable = errors.New("could not look up landmarks, try again")
	ErrNoLandmark           = errors.New("no landmark found at the edge of this playground, choose another location")
	ErrJoinCodeTaken        = errors.New("join code already in use")
)

var idPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func randomHex(bytes int) string {
	value := make([]byte, bytes)
	_, _ = rand.Read(value) // Never fails since Go 1.24; crashes the program instead.
	return hex.EncodeToString(value)
}

type SessionID struct{ value string }

func NewSessionID() SessionID { return SessionID{value: randomHex(16)} }

// ParseSessionID accepts any string-based type, e.g. a wire-format ID.
func ParseSessionID[S ~string](id S) (SessionID, error) {
	if !idPattern.MatchString(string(id)) {
		return SessionID{}, &ValidationError{Message: "session ID must contain 32 lowercase hexadecimal characters"}
	}
	return SessionID{value: string(id)}, nil
}

func (id SessionID) String() string { return id.value }

type PlayerID struct{ value string }

func NewPlayerID() PlayerID { return PlayerID{value: randomHex(16)} }

// ParsePlayerID accepts any string-based type, e.g. a wire-format ID.
func ParsePlayerID[S ~string](id S) (PlayerID, error) {
	if !idPattern.MatchString(string(id)) {
		return PlayerID{}, &ValidationError{Message: "player ID must contain 32 lowercase hexadecimal characters"}
	}
	return PlayerID{value: string(id)}, nil
}

func (id PlayerID) String() string { return id.value }

type Game struct {
	SessionID  SessionID
	CreatedAt  time.Time
	Playground Playground
	JoinCodes  JoinCodes
	Route      Route
	// StartsAt is zero while the lobby is open, otherwise the scheduled start.
	StartsAt time.Time
}
