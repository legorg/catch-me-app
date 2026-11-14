package game

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

func newLobby() Lobby {
	center, _ := ParseCoordinates(52.52, 13.405)
	playground, _ := NewPlayground(center, DefaultCellSize())
	return Lobby{Game: Game{SessionID: NewSessionID(), JoinCodes: NewJoinCodes(), Playground: playground}}
}

// moveToStart reports a position in the runner's start cell.
func moveToStart(t *testing.T, l *Lobby, id PlayerID, now time.Time) {
	t.Helper()
	p, _ := l.Player(id)
	if p.Route.IsZero() {
		return
	}
	position, _ := NewPosition(l.Game.Playground.CellCenter(p.Route.Start), 5, now)
	if err := l.UpdatePosition(id, position, now, false); err != nil {
		t.Fatal(err)
	}
}

// join adds a player; runners immediately stand in their start cell.
func join(t *testing.T, l *Lobby, team Team, name string, now time.Time) Player {
	t.Helper()
	parsed, err := ParsePlayerName(name)
	if err != nil {
		t.Fatal(err)
	}
	player, err := l.Join(l.Game.JoinCodes.For(team), parsed, NewPlayerToken(), now)
	if err != nil {
		t.Fatal(err)
	}
	moveToStart(t, l, player.ID, now)
	player, _ = l.Player(player.ID)
	return player
}

func TestCountdownStartsWhenAllReadyAndBothTeamsPresent(t *testing.T) {
	l := newLobby()
	runner := join(t, &l, Runners, "Ada", start)
	if err := l.SetReady(runner.ID, true, start, false); err != nil || l.Game.Status(start) != StatusLobby {
		t.Fatal("countdown must not start without hunters")
	}
	hunter := join(t, &l, Hunters, "Bob", start)
	if err := l.SetReady(hunter.ID, true, start.Add(time.Second), false); err != nil {
		t.Fatal(err)
	}
	if want := start.Add(time.Second + CountdownDuration); !l.Game.StartsAt.Equal(want) {
		t.Fatalf("starts at %v, want %v", l.Game.StartsAt, want)
	}
	if l.Game.Status(start.Add(2*time.Second)) != StatusCountdown || l.Game.Status(l.Game.StartsAt) != StatusRunning {
		t.Fatal("status not derived from start time")
	}
	// Repeating ready does not restart a running countdown.
	before := l.Game.StartsAt
	if err := l.SetReady(hunter.ID, true, start.Add(5*time.Second), false); err != nil || !l.Game.StartsAt.Equal(before) {
		t.Fatal("countdown restarted")
	}
}

func TestCountdownCancelledAndRestarted(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cancel func(*Lobby, Player, time.Time) error
	}{
		{"unready", func(l *Lobby, p Player, now time.Time) error { return l.SetReady(p.ID, false, now, false) }},
		{"join", func(l *Lobby, _ Player, now time.Time) error {
			name, _ := ParsePlayerName("Late")
			_, err := l.Join(l.Game.JoinCodes.Runner, name, NewPlayerToken(), now)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLobby()
			runner := join(t, &l, Runners, "Ada", start)
			hunter := join(t, &l, Hunters, "Bob", start)
			_ = l.SetReady(runner.ID, true, start, false)
			_ = l.SetReady(hunter.ID, true, start, false)
			if err := tc.cancel(&l, hunter, start.Add(3*time.Second)); err != nil {
				t.Fatal(err)
			}
			if l.Game.Status(start.Add(3*time.Second)) != StatusLobby {
				t.Fatal("countdown not cancelled")
			}
		})
	}
	l := newLobby()
	runner := join(t, &l, Runners, "Ada", start)
	hunter := join(t, &l, Hunters, "Bob", start)
	late := join(t, &l, Hunters, "Cy", start)
	_ = l.SetReady(runner.ID, true, start, false)
	_ = l.SetReady(hunter.ID, true, start, false)
	if err := l.Leave(late.ID, start.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	if want := start.Add(4*time.Second + CountdownDuration); !l.Game.StartsAt.Equal(want) {
		t.Fatal("leaving the only unready player must start a fresh countdown")
	}
}

func TestActionsRejectedOnceRunning(t *testing.T) {
	l := newLobby()
	runner := join(t, &l, Runners, "Ada", start)
	hunter := join(t, &l, Hunters, "Bob", start)
	_ = l.SetReady(runner.ID, true, start, false)
	_ = l.SetReady(hunter.ID, true, start, false)
	running := start.Add(CountdownDuration)
	name, _ := ParsePlayerName("Late")
	_, joinErr := l.Join(l.Game.JoinCodes.Runner, name, NewPlayerToken(), running)
	for _, err := range []error{joinErr, l.SetReady(runner.ID, false, running, false), l.Leave(runner.ID, running)} {
		var rule *RuleError
		if !errors.As(err, &rule) {
			t.Fatalf("expected rule error, got %v", err)
		}
	}
}

func TestJoinRules(t *testing.T) {
	l := newLobby()
	join(t, &l, Runners, "Ada", start)
	var rule *RuleError
	duplicate, _ := ParsePlayerName("ADA")
	if _, err := l.Join(l.Game.JoinCodes.Hunter, duplicate, NewPlayerToken(), start); !errors.As(err, &rule) {
		t.Fatal("duplicate name accepted")
	}
	unknown, _ := ParseJoinCode("ZZZZZZ")
	if _, err := l.Join(unknown, duplicate, NewPlayerToken(), start); !errors.Is(err, ErrNotFound) {
		t.Fatal("unknown code accepted")
	}
	for i := len(l.Players); i < MaxPlayers; i++ {
		join(t, &l, Hunters, "P"+strings.Repeat("x", i%10)+string(rune('a'+i)), start)
	}
	extra, _ := ParsePlayerName("Extra")
	if _, err := l.Join(l.Game.JoinCodes.Runner, extra, NewPlayerToken(), start); !errors.As(err, &rule) {
		t.Fatal("full game accepted a player")
	}
}

func TestPlayerParsing(t *testing.T) {
	codes := NewJoinCodes()
	if codes.Runner == codes.Hunter {
		t.Fatal("codes must differ")
	}
	if code, err := ParseJoinCode(" " + strings.ToLower(codes.Runner.String()) + " "); err != nil || code != codes.Runner {
		t.Fatal("join codes must be case-insensitive and trimmed")
	}
	if team, ok := codes.TeamFor(codes.Hunter); !ok || team != Hunters {
		t.Fatal("hunter code not recognized")
	}
	var validation *ValidationError
	for _, code := range []string{"", "ABCDE", "ABCDEFG", "ABCDE0", "ABCDE1", "ABCDEO", "ABC-EF"} {
		if _, err := ParseJoinCode(code); !errors.As(err, &validation) {
			t.Fatalf("invalid code accepted: %q", code)
		}
	}
	if name, err := ParsePlayerName("  Zoë  "); err != nil || name.String() != "Zoë" {
		t.Fatal("name not trimmed")
	}
	for _, name := range []string{"", "   ", strings.Repeat("a", 21), "a\nb", "\xff"} {
		if _, err := ParsePlayerName(name); !errors.As(err, &validation) {
			t.Fatalf("invalid name accepted: %q", name)
		}
	}
	if _, err := ParsePlayerName(strings.Repeat("ü", 20)); err != nil {
		t.Fatal("name length must count characters, not bytes")
	}
	token := NewPlayerToken()
	if parsed, err := ParsePlayerToken(token.String()); err != nil || parsed.Hash() != token.Hash() || token.Hash().String() == token.String() {
		t.Fatal("token round trip or hash failed")
	}
	if _, err := ParsePlayerToken("short"); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("invalid token must be unauthorized")
	}
	if _, err := ParseTeam("gamemaster"); !errors.As(err, &validation) {
		t.Fatal("unknown team accepted")
	}
}
