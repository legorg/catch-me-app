package game

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	JoinCodeLength = 6
	MaxNameLength  = 20
	// Uppercase letters and digits without easily confused 0/O and 1/I.
	joinCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

type Team string

const (
	Runners Team = "runner"
	Hunters Team = "hunter"
)

func ParseTeam(team string) (Team, error) {
	switch Team(team) {
	case Runners, Hunters:
		return Team(team), nil
	}
	return "", &ValidationError{Message: "team must be runner or hunter"}
}

type JoinCode struct{ value string }

func NewJoinCode() JoinCode {
	code := make([]byte, JoinCodeLength)
	_, _ = rand.Read(code)
	for i, b := range code {
		// 256 is a multiple of the 32-character alphabet, so this is unbiased.
		code[i] = joinCodeAlphabet[int(b)%len(joinCodeAlphabet)]
	}
	return JoinCode{value: string(code)}
}

// ParseJoinCode accepts surrounding spaces and lowercase input.
func ParseJoinCode(code string) (JoinCode, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if len(normalized) != JoinCodeLength || strings.Trim(normalized, joinCodeAlphabet) != "" {
		return JoinCode{}, &ValidationError{Message: "join code must contain 6 letters or digits"}
	}
	return JoinCode{value: normalized}, nil
}

func (c JoinCode) String() string { return c.value }

type JoinCodes struct{ Runner, Hunter JoinCode }

func NewJoinCodes() JoinCodes {
	codes := JoinCodes{Runner: NewJoinCode(), Hunter: NewJoinCode()}
	for codes.Hunter == codes.Runner {
		codes.Hunter = NewJoinCode()
	}
	return codes
}

func (c JoinCodes) For(team Team) JoinCode {
	if team == Hunters {
		return c.Hunter
	}
	return c.Runner
}

func (c JoinCodes) TeamFor(code JoinCode) (Team, bool) {
	switch code {
	case c.Runner:
		return Runners, true
	case c.Hunter:
		return Hunters, true
	}
	return "", false
}

type PlayerName struct{ value string }

func ParsePlayerName(name string) (PlayerName, error) {
	trimmed := strings.TrimSpace(name)
	length := utf8.RuneCountInString(trimmed)
	if length == 0 || length > MaxNameLength || !utf8.ValidString(trimmed) || strings.ContainsFunc(trimmed, unicode.IsControl) {
		return PlayerName{}, &ValidationError{Message: "name must contain 1 to 20 characters"}
	}
	return PlayerName{value: trimmed}, nil
}

func (n PlayerName) String() string { return n.value }

// SameAs compares names ignoring case, so players remain distinguishable.
func (n PlayerName) SameAs(other PlayerName) bool { return strings.EqualFold(n.value, other.value) }

var secretPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// PlayerToken is the secret a player authenticates with. Only its hash is stored.
type PlayerToken struct{ value string }

func NewPlayerToken() PlayerToken { return PlayerToken{value: randomHex(32)} }

func ParsePlayerToken(token string) (PlayerToken, error) {
	if !secretPattern.MatchString(token) {
		return PlayerToken{}, ErrUnauthorized
	}
	return PlayerToken{value: token}, nil
}

func (t PlayerToken) String() string { return t.value }

func (t PlayerToken) Hash() TokenHash {
	sum := sha256.Sum256([]byte(t.value))
	return TokenHash{value: hex.EncodeToString(sum[:])}
}

type TokenHash struct{ value string }

func ParseTokenHash(hash string) (TokenHash, error) {
	if !secretPattern.MatchString(hash) {
		return TokenHash{}, &ValidationError{Message: "token hash must contain 64 lowercase hexadecimal characters"}
	}
	return TokenHash{value: hash}, nil
}

func (h TokenHash) String() string { return h.value }

type Player struct {
	ID        PlayerID
	Name      PlayerName
	Team      Team
	Ready     bool
	TokenHash TokenHash
	JoinedAt  time.Time
	// Zero values mean "none": hunters have no route, Position is the latest
	// report, and Reveal is what hunters last saw of a runner.
	Route    RunnerRoute
	Position Position
	Reveal   Reveal
}
