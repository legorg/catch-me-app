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

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

var ErrNotFound = errors.New("game not found")

type SessionID struct{ value string }

var sessionIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func NewSessionID() SessionID {
	var id [16]byte
	_, _ = rand.Read(id[:]) // Never fails since Go 1.24; crashes the program instead.
	return SessionID{value: hex.EncodeToString(id[:])}
}

func ParseSessionID(id string) (SessionID, error) {
	if !sessionIDPattern.MatchString(id) {
		return SessionID{}, &ValidationError{Message: "session ID must contain 32 lowercase hexadecimal characters"}
	}
	return SessionID{value: id}, nil
}

func (id SessionID) String() string { return id.value }

type Game struct {
	SessionID  SessionID
	CreatedAt  time.Time
	Playground Playground
}
