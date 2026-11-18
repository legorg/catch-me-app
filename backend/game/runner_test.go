package game

import (
	"errors"
	"testing"
	"time"
)

func TestRunnerStateValidation(t *testing.T) {
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.FixedZone("test", 3600))
	for _, tc := range []struct {
		status RunnerStatus
		at     time.Time
		valid  bool
	}{
		{RunnerActive, time.Time{}, true},
		{RunnerCaught, at, true},
		{RunnerFinished, at, true},
		{RunnerActive, at, false},
		{RunnerCaught, time.Time{}, false},
		{RunnerFinished, time.Time{}, false},
		{"unknown", at, false},
		{"left", at, false}, // Leaving a running game is not implemented yet.
		{"", time.Time{}, false},
	} {
		s, err := ParseRunnerState(string(tc.status), tc.at)
		if (err == nil) != tc.valid {
			t.Fatalf("parse %s/%v: %v", tc.status, tc.at, err)
		}
		if tc.valid && (s.Status() != tc.status || !s.EndedAt().Equal(tc.at)) {
			t.Fatalf("state changed: %+v", s)
		}
		if tc.valid && !tc.at.IsZero() && s.EndedAt().Location() != time.UTC {
			t.Fatal("end time must be normalized to UTC")
		}
	}
	var zero RunnerState
	if zero.Status() != RunnerActive || !zero.EndedAt().IsZero() {
		t.Fatal("zero state must be active")
	}
}

func TestRunnerOutcomeTransitions(t *testing.T) {
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	for _, outcome := range []RunnerStatus{RunnerCaught, RunnerFinished} {
		var s RunnerState
		if err := s.end(outcome, at); err != nil {
			t.Fatal(err)
		}
		if err := s.end(outcome, at.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if s.Status() != outcome || s.EndedAt() != at {
			t.Fatal("retry changed outcome/time")
		}
		for _, next := range []RunnerStatus{RunnerActive, RunnerCaught, RunnerFinished} {
			if next == outcome {
				continue
			}
			var rule *RuleError
			if err := s.end(next, at); !errors.As(err, &rule) {
				t.Fatalf("terminal transition %s -> %s: %v", outcome, next, err)
			}
			if s.Status() != outcome || s.EndedAt() != at {
				t.Fatal("rejected transition changed state")
			}
		}
		if !s.ActiveAt(at.Add(-time.Nanosecond)) || s.ActiveAt(at) || s.ActiveAt(at.Add(time.Nanosecond)) {
			t.Fatal("historical eligibility must stop at the exact end time")
		}
	}
	var s RunnerState
	if err := s.end(RunnerCaught, time.Time{}); err == nil || s.Status() != RunnerActive {
		t.Fatal("ending without a timestamp must not change state")
	}
}
