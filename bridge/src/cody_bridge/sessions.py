"""States, sessions and merging (BRG-02, BRG-04; spec/architecture.md 4.1)."""

from __future__ import annotations

import enum
from dataclasses import dataclass


class State(enum.Enum):
    IDLE = "IDLE"
    WORKING = "WORKING"
    THINKING = "THINKING"
    APPROVAL = "APPROVAL"
    COMPLETED = "COMPLETED"
    ERROR = "ERROR"


# Most urgent first (BRG-02)
PRIORITY = [
    State.APPROVAL,
    State.ERROR,
    State.WORKING,
    State.THINKING,
    State.COMPLETED,
    State.IDLE,
]

# States a session is dropped from after the timeout (BRG-04)
EXPIRING = {State.WORKING, State.THINKING, State.APPROVAL}


@dataclass
class Session:
    agent: str
    session: str
    state: State
    last_event: float  # monotonic seconds


class Sessions:
    """Tracks the state per (agent, session) and merges them."""

    def __init__(self, timeout_s: float) -> None:
        self.timeout_s = timeout_s
        self._sessions: dict[tuple[str, str], Session] = {}

    def update(self, agent: str, session: str, state: State, now: float) -> None:
        self._sessions[(agent, session)] = Session(agent, session, state, now)

    def end(self, agent: str, session: str) -> None:
        self._sessions.pop((agent, session), None)

    def clear(self) -> None:
        """Drops all sessions, including the manual one (BRG-15)."""
        self._sessions.clear()

    def expire(self, now: float) -> None:
        """Drops sessions in WORKING, THINKING or APPROVAL without events for the timeout."""
        for key, s in list(self._sessions.items()):
            if s.state in EXPIRING and now - s.last_event >= self.timeout_s:
                del self._sessions[key]

    def merged(self) -> State:
        """The most urgent state of all sessions; IDLE without sessions."""
        states = {s.state for s in self._sessions.values()}
        return next((p for p in PRIORITY if p in states), State.IDLE)

    def all(self) -> list[Session]:
        return list(self._sessions.values())
