package db

import (
	"fmt"

	"catch-me/backend/game"
)

func toJokerSettingsRow(s game.JokerSettings) jokerSettingsRow {
	pointer := func(a game.JokerAllowance) *int {
		uses := a.Uses()
		return &uses
	}
	return jokerSettingsRow{
		ShowRunnersAllowance: pointer(s.ShowRunners), QueryZoneAllowance: pointer(s.QueryZone),
		CompassAllowance: pointer(s.Compass), TrapAllowance: pointer(s.Trap.JokerAllowance),
		ZoneOnlyAllowance: pointer(s.ZoneOnly), FakePositionAllowance: pointer(s.FakePosition),
		ShowHuntersAllowance: pointer(s.ShowHunters),
	}
}

func parseJokerSettings(row jokerSettingsRow) (game.JokerSettings, error) {
	var settings game.JokerSettings
	fields := []struct {
		kind   game.JokerKind
		stored *int
		target *game.JokerAllowance
	}{
		{game.JokerShowRunners, row.ShowRunnersAllowance, &settings.ShowRunners},
		{game.JokerQueryZone, row.QueryZoneAllowance, &settings.QueryZone},
		{game.JokerCompass, row.CompassAllowance, &settings.Compass},
		{game.JokerTrap, row.TrapAllowance, &settings.Trap.JokerAllowance},
		{game.JokerZoneOnly, row.ZoneOnlyAllowance, &settings.ZoneOnly},
		{game.JokerFakePosition, row.FakePositionAllowance, &settings.FakePosition},
		{game.JokerShowHunters, row.ShowHuntersAllowance, &settings.ShowHunters},
	}
	for _, field := range fields {
		if field.stored == nil {
			return game.JokerSettings{}, fmt.Errorf("missing %s allowance", field.kind)
		}
		allowance, err := game.ParseJokerAllowance(*field.stored)
		if err != nil {
			return game.JokerSettings{}, fmt.Errorf("%s allowance: %v", field.kind, err)
		}
		*field.target = allowance
	}
	return settings, nil
}
