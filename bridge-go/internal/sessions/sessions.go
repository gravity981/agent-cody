package sessions

import (
	"math"
	"sort"
	"time"
)

type State string

const (
	IDLE      State = "IDLE"
	WORKING   State = "WORKING"
	THINKING  State = "THINKING"
	APPROVAL  State = "APPROVAL"
	COMPLETED State = "COMPLETED"
	ERROR     State = "ERROR"
)

var priority = [...]State{APPROVAL, ERROR, WORKING, THINKING, COMPLETED, IDLE}

func ParseState(value string) (State, bool) {
	for _, state := range priority {
		if value == string(state) {
			return state, true
		}
	}
	return "", false
}

type Key struct {
	Agent   string
	Session string
}

type Session struct {
	Agent     string
	ID        string
	State     State
	LastEvent time.Time
}

type Sessions struct {
	timeout time.Duration
	items   map[Key]Session
}

func New(timeout time.Duration) *Sessions {
	return &Sessions{timeout: timeout, items: make(map[Key]Session)}
}

func (s *Sessions) Update(agent, id string, state State, now time.Time) {
	s.items[Key{Agent: agent, Session: id}] = Session{
		Agent: agent, ID: id, State: state, LastEvent: now,
	}
}

func (s *Sessions) End(agent, id string) {
	delete(s.items, Key{Agent: agent, Session: id})
}

func (s *Sessions) Clear() {
	clear(s.items)
}

func (s *Sessions) Expire(now time.Time) {
	for key, session := range s.items {
		switch session.State {
		case WORKING, THINKING, APPROVAL:
			if now.Sub(session.LastEvent) >= s.timeout {
				delete(s.items, key)
			}
		}
	}
}

func (s *Sessions) Merged() State {
	found := make(map[State]struct{}, len(s.items))
	for _, session := range s.items {
		found[session.State] = struct{}{}
	}
	for _, state := range priority {
		if _, ok := found[state]; ok {
			return state
		}
	}
	return IDLE
}

func (s *Sessions) All() []Session {
	result := make([]Session, 0, len(s.items))
	for _, session := range s.items {
		result = append(result, session)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Agent != result[j].Agent {
			return result[i].Agent < result[j].Agent
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func (s Session) IdleSeconds(now time.Time) int {
	return int(math.Round(now.Sub(s.LastEvent).Seconds()))
}
