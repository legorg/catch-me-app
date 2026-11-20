package game

// JokerAllowance is the configured number of uses, not the remaining uses.
// Its zero value disables the joker; defaults must be chosen explicitly.
type JokerAllowance struct{ uses int }

func ParseJokerAllowance(uses int) (JokerAllowance, error) {
	if uses < 0 || uses > 5 {
		return JokerAllowance{}, &ValidationError{Message: "joker allowance must be between 0 and 5"}
	}
	return JokerAllowance{uses: uses}, nil
}

func (a JokerAllowance) Uses() int { return a.uses }

// TrapSettings can grow independently of the settings of other joker kinds.
// Radius and activation delay currently remain fixed gameplay constants.
type TrapSettings struct{ JokerAllowance }

// JokerSettings has exactly one setting per implemented kind. Settings belong
// to the game and stay unchanged after creation; uses retain their existing
// team or personal ownership.
type JokerSettings struct {
	ShowRunners  JokerAllowance
	QueryZone    JokerAllowance
	Compass      JokerAllowance
	Trap         TrapSettings
	ZoneOnly     JokerAllowance
	FakePosition JokerAllowance
	ShowHunters  JokerAllowance
}

func DefaultJokerSettings() JokerSettings {
	a := JokerAllowance{uses: 1}
	return JokerSettings{
		ShowRunners: a, QueryZone: a, Compass: a, Trap: TrapSettings{JokerAllowance: a},
		ZoneOnly: a, FakePosition: a, ShowHunters: a,
	}
}

// Allowance returns zero for unknown kinds; settings never grant access to a
// joker outside the player's permitted kinds.
func (s JokerSettings) Allowance(kind JokerKind) JokerAllowance {
	switch kind {
	case JokerShowRunners:
		return s.ShowRunners
	case JokerQueryZone:
		return s.QueryZone
	case JokerCompass:
		return s.Compass
	case JokerTrap:
		return s.Trap.JokerAllowance
	case JokerZoneOnly:
		return s.ZoneOnly
	case JokerFakePosition:
		return s.FakePosition
	case JokerShowHunters:
		return s.ShowHunters
	default:
		return JokerAllowance{}
	}
}
