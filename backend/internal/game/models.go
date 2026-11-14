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
	ErrJoinCodeTaken = errors.New("join code already in use")
)

var idPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func randomHex(bytes int) string {
	value := make([]byte, bytes)
	_, _ = rand.Read(value) // Never fails since Go 1.24; crashes the program instead.
	return hex.EncodeToString(value)
}

type SessionID struct{ value string }

func NewSessionID() SessionID { return SessionID{value: randomHex(16)} }

func ParseSessionID(id string) (SessionID, error) {
	if !idPattern.MatchString(id) {
		return SessionID{}, &ValidationError{Message: "session ID must contain 32 lowercase hexadecimal characters"}
	}
	return SessionID{value: id}, nil
}

func (id SessionID) String() string { return id.value }

type PlayerID struct{ value string }

func NewPlayerID() PlayerID { return PlayerID{value: randomHex(16)} }

func ParsePlayerID(id string) (PlayerID, error) {
	if !idPattern.MatchString(id) {
		return PlayerID{}, &ValidationError{Message: "player ID must contain 32 lowercase hexadecimal characters"}
	}
	return PlayerID{value: id}, nil
}

func (id PlayerID) String() string { return id.value }

type Game struct {
	SessionID  SessionID
	CreatedAt  time.Time
	Playground Playground
	JoinCodes  JoinCodes
	// StartsAt is zero while the lobby is open, otherwise the scheduled start.
	StartsAt time.Time
}
