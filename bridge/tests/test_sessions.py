"""Session merging and timeouts (BRG-02, BRG-04)."""

from __future__ import annotations

import pytest
from cody_bridge.sessions import PRIORITY, Sessions, State


def test_req_BRG_02_no_sessions_is_idle() -> None:
    assert Sessions(600).merged() == State.IDLE


@pytest.mark.parametrize("higher,lower", list(zip(PRIORITY, PRIORITY[1:], strict=False)))
def test_req_BRG_02_most_urgent_state_wins(higher: State, lower: State) -> None:
    s = Sessions(600)
    s.update("a", "1", lower, 0)
    s.update("b", "2", higher, 0)
    assert s.merged() == higher


def test_req_BRG_02_order_is_approval_error_working_thinking_completed_idle() -> None:
    assert [p.value for p in PRIORITY] == [
        "APPROVAL",
        "ERROR",
        "WORKING",
        "THINKING",
        "COMPLETED",
        "IDLE",
    ]


def test_req_BRG_02_each_session_keeps_its_own_state() -> None:
    s = Sessions(600)
    s.update("claude-code", "1", State.WORKING, 0)
    s.update("claude-code", "2", State.COMPLETED, 0)
    s.update("claude-code", "1", State.COMPLETED, 1)  # session 1 finishes
    assert {x.session: x.state for x in s.all()} == {"1": State.COMPLETED, "2": State.COMPLETED}
    assert s.merged() == State.COMPLETED


@pytest.mark.parametrize("state", [State.WORKING, State.THINKING, State.APPROVAL])
def test_req_BRG_04_active_sessions_are_dropped_after_the_timeout(state: State) -> None:
    s = Sessions(600)
    s.update("a", "1", state, 0)
    s.expire(599)
    assert s.merged() == state
    s.expire(600)
    assert s.all() == []
    assert s.merged() == State.IDLE


@pytest.mark.parametrize("state", [State.COMPLETED, State.ERROR])
def test_req_BRG_04_finished_sessions_stay_until_next_event_or_end(state: State) -> None:
    s = Sessions(600)
    s.update("a", "1", state, 0)
    s.expire(100_000)
    assert s.merged() == state
    s.update("a", "1", State.THINKING, 100_001)  # next event
    assert s.merged() == State.THINKING
    s.update("a", "1", state, 100_002)
    s.end("a", "1")  # its end
    assert s.all() == []


def test_req_BRG_04_timeout_is_configurable_and_counts_from_the_last_event() -> None:
    s = Sessions(60)
    s.update("a", "1", State.WORKING, 0)
    s.update("a", "1", State.WORKING, 50)  # new event restarts the clock
    s.expire(109)
    assert s.merged() == State.WORKING
    s.expire(110)
    assert s.merged() == State.IDLE
