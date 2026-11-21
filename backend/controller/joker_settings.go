package controller

import (
	"fmt"

	"catch-me/backend/game"
)

// JokerSetting is one kind's setting in creation requests and responses.
// The pointer distinguishes a missing/null request value from an explicit 0.
type JokerSetting struct {
	Allowance *int `json:"allowance" minimum:"0" maximum:"5" example:"1"`
}

// TrapSetting can gain kind-specific fields without changing other kinds.
type TrapSetting struct{ JokerSetting }

// ZoneAlertSetting switches the passive zone alert, which has no allowance.
// The pointer distinguishes a missing/null request value from false.
type ZoneAlertSetting struct {
	Enabled *bool `json:"enabled" example:"true"`
}

// JokerSettings is the game's configuration, not team or personal usage history.
type JokerSettings struct {
	ShowRunners  *JokerSetting     `json:"showRunners"`
	QueryZone    *JokerSetting     `json:"queryZone"`
	Compass      *JokerSetting     `json:"compass"`
	Trap         *TrapSetting      `json:"trap"`
	ZoneOnly     *JokerSetting     `json:"zoneOnly"`
	FakePosition *JokerSetting     `json:"fakePosition"`
	ShowHunters  *JokerSetting     `json:"showHunters"`
	ZoneAlert    *ZoneAlertSetting `json:"zoneAlert"`
}

func jokerSettingsResponse(s game.JokerSettings) JokerSettings {
	setting := func(a game.JokerAllowance) *JokerSetting {
		allowance := a.Max()
		return &JokerSetting{Allowance: &allowance}
	}
	enabled := s.ZoneAlert.Enabled
	return JokerSettings{
		ShowRunners: setting(s.ShowRunners), QueryZone: setting(s.QueryZone), Compass: setting(s.Compass),
		Trap: &TrapSetting{JokerSetting: *setting(s.Trap.JokerAllowance)}, ZoneOnly: setting(s.ZoneOnly),
		FakePosition: setting(s.FakePosition), ShowHunters: setting(s.ShowHunters),
		ZoneAlert: &ZoneAlertSetting{Enabled: &enabled},
	}
}

func (body *JokerSettings) parse() (game.JokerSettings, error) {
	if body == nil {
		return game.JokerSettings{}, &game.ValidationError{Message: "jokers are required"}
	}
	var settings game.JokerSettings
	if body.ZoneAlert == nil || body.ZoneAlert.Enabled == nil {
		return game.JokerSettings{}, &game.ValidationError{Message: "zoneAlert enabled is required"}
	}
	settings.ZoneAlert.Enabled = *body.ZoneAlert.Enabled
	var trap *JokerSetting
	if body.Trap != nil {
		trap = &body.Trap.JokerSetting
	}
	fields := []struct {
		kind   game.JokerKind
		input  *JokerSetting
		target *game.JokerAllowance
	}{
		{game.JokerShowRunners, body.ShowRunners, &settings.ShowRunners},
		{game.JokerQueryZone, body.QueryZone, &settings.QueryZone},
		{game.JokerCompass, body.Compass, &settings.Compass},
		{game.JokerTrap, trap, &settings.Trap.JokerAllowance},
		{game.JokerZoneOnly, body.ZoneOnly, &settings.ZoneOnly},
		{game.JokerFakePosition, body.FakePosition, &settings.FakePosition},
		{game.JokerShowHunters, body.ShowHunters, &settings.ShowHunters},
	}
	for _, field := range fields {
		if field.input == nil || field.input.Allowance == nil {
			return game.JokerSettings{}, &game.ValidationError{Message: fmt.Sprintf("%s allowance is required", field.kind)}
		}
		allowance, err := game.ParseJokerAllowance(*field.input.Allowance)
		if err != nil {
			return game.JokerSettings{}, &game.ValidationError{Message: fmt.Sprintf("%s: %s", field.kind, err)}
		}
		*field.target = allowance
	}
	return settings, nil
}
