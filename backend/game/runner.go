package game

import "time"

type RunnerStatus string

const (
	RunnerActive   RunnerStatus = "active"
	RunnerCaught   RunnerStatus = "caught"
	RunnerFinished RunnerStatus = "finished"
)

// RunnerState has exactly one outcome and completion time. Its zero value is
// active; private fields keep contradictory outcomes out of the domain model.
// It is meaningful only for players whose original team is Runners.
type RunnerState struct {
	status  RunnerStatus
	endedAt time.Time
}

func ParseRunnerState(status string, endedAt time.Time) (RunnerState, error) {
	switch RunnerStatus(status) {
	case RunnerActive:
		if endedAt.IsZero() {
			return RunnerState{}, nil
		}
	case RunnerCaught, RunnerFinished:
		if !endedAt.IsZero() {
			return RunnerState{status: RunnerStatus(status), endedAt: endedAt.UTC()}, nil
		}
	}
	return RunnerState{}, &ValidationError{Message: "runner state must be active without an end time, or caught/finished with an end time"}
}

func (s RunnerState) Status() RunnerStatus {
	if s.status == "" {
		return RunnerActive
	}
	return s.status
}

func (s RunnerState) EndedAt() time.Time { return s.endedAt }

// ActiveAt preserves historical joker eligibility, including at the exact end time.
func (s RunnerState) ActiveAt(at time.Time) bool {
	return s.Status() == RunnerActive || s.endedAt.After(at)
}

// end permits only active -> terminal transitions; same-outcome retries are harmless.
func (s *RunnerState) end(status RunnerStatus, at time.Time) error {
	if s.Status() == status {
		return nil
	}
	if s.Status() != RunnerActive {
		return &RuleError{Message: "a runner's outcome cannot be changed"}
	}
	next, err := ParseRunnerState(string(status), at)
	if err != nil {
		return err
	}
	*s = next
	return nil
}
