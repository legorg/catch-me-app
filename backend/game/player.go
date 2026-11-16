package game

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	joinCodeLength = 6
	maxNameLength  = 20
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

// NewJoinCode returns 6 random digits; leading zeros are kept.
func NewJoinCode() JoinCode {
	n, _ := rand.Int(rand.Reader, big.NewInt(1_000_000)) // Never fails since Go 1.24; crashes the program instead.
	return JoinCode{value: fmt.Sprintf("%0*d", joinCodeLength, n.Int64())}
}

// ParseJoinCode accepts surrounding spaces.
func ParseJoinCode(code string) (JoinCode, error) {
	normalized := strings.TrimSpace(code)
	if len(normalized) != joinCodeLength || strings.Trim(normalized, "0123456789") != "" {
		return JoinCode{}, &ValidationError{Message: "join code must contain 6 digits"}
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

func (c JoinCodes) TeamFor(code JoinCode) (Team, bool) {
	switch code {
	case c.Runner:
		return Runners, true
	case c.Hunter:
		return Hunters, true
	}
	return "", false
}

// parseName trims a player or team name and checks its length and characters.
func parseName(name, what string) (string, error) {
	trimmed := strings.TrimSpace(name)
	length := utf8.RuneCountInString(trimmed)
	if length == 0 || length > maxNameLength || !utf8.ValidString(trimmed) || strings.ContainsFunc(trimmed, unicode.IsControl) {
		return "", &ValidationError{Message: what + " must contain 1 to 20 characters"}
	}
	return trimmed, nil
}

type PlayerName struct{ value string }

func ParsePlayerName(name string) (PlayerName, error) {
	trimmed, err := parseName(name, "name")
	return PlayerName{value: trimmed}, err
}

func (n PlayerName) String() string { return n.value }

// sameAs compares names ignoring case, so players remain distinguishable.
func (n PlayerName) sameAs(other PlayerName) bool { return strings.EqualFold(n.value, other.value) }

// HunterTeamName names a group of hunters who see each other's positions. A
// hunter team exists exactly while at least one hunter belongs to it.
type HunterTeamName struct{ value string }

func ParseHunterTeamName(name string) (HunterTeamName, error) {
	trimmed, err := parseName(name, "team name")
	return HunterTeamName{value: trimmed}, err
}

func (n HunterTeamName) String() string { return n.value }
func (n HunterTeamName) IsZero() bool   { return n.value == "" }

// sameAs compares team names ignoring case.
func (n HunterTeamName) sameAs(other HunterTeamName) bool {
	return strings.EqualFold(n.value, other.value)
}

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
	// HunterTeam is zero for runners and for hunters who have not chosen one.
	HunterTeam HunterTeamName
	// Zero values mean "none": hunters have no route; Position is the latest
	// report. Reveals live on the Lobby.
	Route    RunnerRoute
	Position Position
}
