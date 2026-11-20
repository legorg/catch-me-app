package game

// JokerAllowance is how often a joker may be used in total, not how often it
// is left. Its zero value disables the joker; defaults must be chosen explicitly.
type JokerAllowance struct{ max int }

func ParseJokerAllowance(max int) (JokerAllowance, error) {
	if max < 0 || max > 5 {
		return JokerAllowance{}, &ValidationError{Message: "joker allowance must be between 0 and 5"}
	}
	return JokerAllowance{max: max}, nil
}

func (a JokerAllowance) Max() int { return a.max }

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

// allowances is the only mapping from kinds to their allowance fields.
func (s *JokerSettings) allowances() map[JokerKind]*JokerAllowance {
	return map[JokerKind]*JokerAllowance{
		JokerShowRunners: &s.ShowRunners, JokerQueryZone: &s.QueryZone, JokerCompass: &s.Compass,
		JokerTrap: &s.Trap.JokerAllowance, JokerZoneOnly: &s.ZoneOnly,
		JokerFakePosition: &s.FakePosition, JokerShowHunters: &s.ShowHunters,
	}
}

func DefaultJokerSettings() JokerSettings {
	var s JokerSettings
	for _, a := range s.allowances() {
		a.max = 1
	}
	return s
}

// Allowance returns zero for unknown kinds; settings never grant access to a
// joker outside the player's permitted kinds.
func (s JokerSettings) Allowance(kind JokerKind) JokerAllowance {
	if a, ok := s.allowances()[kind]; ok {
		return *a
	}
	return JokerAllowance{}
}
