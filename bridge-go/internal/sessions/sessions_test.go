package sessions

import (
	"testing"
	"time"
)

func Test_req_BRG_02_MergesByPriorityAndKeepsSessionsSeparate(t *testing.T) {
	now := time.Unix(100, 0)
	active := New(10 * time.Minute)
	if active.Merged() != IDLE {
		t.Fatal("empty session set did not merge to IDLE")
	}
	active.Update("claude-code", "one", COMPLETED, now)
	active.Update("claude-code", "two", WORKING, now)
	active.Update("copilot", "one", APPROVAL, now)

	if got := active.Merged(); got != APPROVAL {
		t.Fatalf("merged state = %s, want APPROVAL", got)
	}
	if got := len(active.All()); got != 3 {
		t.Fatalf("session count = %d, want 3", got)
	}
	active.End("copilot", "one")
	if got := active.Merged(); got != WORKING {
		t.Fatalf("merged state after ending approval = %s, want WORKING", got)
	}
}

func Test_req_BRG_02_EveryHigherStateWinsItsLowerNeighbor(t *testing.T) {
	for i := 0; i < len(priority)-1; i++ {
		active := New(time.Minute)
		now := time.Unix(100, 0)
		active.Update("lower", "session", priority[i+1], now)
		active.Update("higher", "session", priority[i], now)
		if got := active.Merged(); got != priority[i] {
			t.Errorf("%s did not outrank %s; merged = %s", priority[i], priority[i+1], got)
		}
	}
}

func Test_req_BRG_04_ExpiresOnlyActiveStatesFromTheirLastEvent(t *testing.T) {
	start := time.Unix(100, 0)
	active := New(time.Minute)
	active.Update("agent", "working", WORKING, start)
	active.Update("agent", "thinking", THINKING, start)
	active.Update("agent", "approval", APPROVAL, start)
	active.Update("agent", "completed", COMPLETED, start)
	active.Update("agent", "error", ERROR, start)
	active.Update("agent", "recent", WORKING, start.Add(30*time.Second))

	active.Expire(start.Add(80 * time.Second))
	if got := len(active.All()); got != 3 {
		t.Fatalf("session count after expiry = %d, want 3", got)
	}
	if got := active.Merged(); got != ERROR {
		t.Fatalf("merged state after expiry = %s, want ERROR", got)
	}
	active.Expire(start.Add(90*time.Second + time.Nanosecond))
	if got := len(active.All()); got != 2 {
		t.Fatalf("session count after recent session expires = %d, want 2", got)
	}
	if got := active.Merged(); got != ERROR {
		t.Fatalf("finished sessions did not persist; merged state = %s, want ERROR", got)
	}
}

func Test_req_BRG_15_ClearRemovesAllSessions(t *testing.T) {
	active := New(time.Minute)
	now := time.Unix(100, 0)
	active.Update("claude-code", "one", WORKING, now)
	active.Update("manual", "manual", APPROVAL, now)
	active.Clear()

	if len(active.All()) != 0 || active.Merged() != IDLE {
		t.Fatal("clear did not return the bridge to IDLE")
	}
}
