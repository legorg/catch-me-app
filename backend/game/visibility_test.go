package game

import "testing"

func TestHunterTeamVisibilityIgnoresCaseButRequiresATeam(t *testing.T) {
	pack, _ := ParseHunterTeamName("Pack")
	lowercasePack, _ := ParseHunterTeamName("pack")
	wolves, _ := ParseHunterTeamName("Wolves")
	position, _ := NewPosition(Coordinates{}, 5, start)
	for _, tc := range []struct {
		name             string
		viewerTeam       Team
		viewer, teammate HunterTeamName
		visible          bool
	}{
		{"same team", Hunters, pack, pack, true},
		{"different spelling", Hunters, pack, lowercasePack, true},
		{"different team", Hunters, pack, wolves, false},
		{"both without team", Hunters, HunterTeamName{}, HunterTeamName{}, false},
		{"viewer without team", Hunters, HunterTeamName{}, pack, false},
		{"target without team", Hunters, pack, HunterTeamName{}, false},
		{"runner", Runners, pack, pack, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viewer := Player{ID: NewPlayerID(), Team: tc.viewerTeam, HunterTeam: tc.viewer}
			other := Player{ID: NewPlayerID(), Team: Hunters, HunterTeam: tc.teammate, Position: position}
			l := Lobby{Players: []Player{viewer, other}}
			got := l.VisibleTo(viewer, start)[1].Position
			if tc.visible && got != position || !tc.visible && !got.IsZero() {
				t.Fatalf("teammate position visible = %v, want %v", !got.IsZero(), tc.visible)
			}
		})
	}
}
