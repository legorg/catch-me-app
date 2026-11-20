package controller

import (
	"fmt"

	"catch-me/backend/game"
)

// Pointers distinguish missing/null settings from an explicit allowance of 0.
type JokerAllowanceRequest struct {
	Allowance *int `json:"allowance" minimum:"0" maximum:"5" example:"1"`
}

// Trap settings can gain kind-specific fields without changing other kinds.
type TrapSettingsRequest struct{ JokerAllowanceRequest }

type JokerSettingsRequest struct {
	ShowRunners  *JokerAllowanceRequest `json:"showRunners"`
	QueryZone    *JokerAllowanceRequest `json:"queryZone"`
	Compass      *JokerAllowanceRequest `json:"compass"`
	Trap         *TrapSettingsRequest   `json:"trap"`
	ZoneOnly     *JokerAllowanceRequest `json:"zoneOnly"`
	FakePosition *JokerAllowanceRequest `json:"fakePosition"`
	ShowHunters  *JokerAllowanceRequest `json:"showHunters"`
}

func (body *JokerSettingsRequest) parse() (game.JokerSettings, error) {
	if body == nil {
		return game.JokerSettings{}, &game.ValidationError{Message: "jokers are required"}
	}
	var settings game.JokerSettings
	var trap *JokerAllowanceRequest
	if body.Trap != nil {
		trap = &body.Trap.JokerAllowanceRequest
	}
	fields := []struct {
		kind   game.JokerKind
		input  *JokerAllowanceRequest
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
