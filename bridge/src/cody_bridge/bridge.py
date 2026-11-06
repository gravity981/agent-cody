"""The bridge core: sessions, merged state and the device (BRG-01 – BRG-04, BRG-12).

Thread-safe: HTTP threads call `event`, `set_manual`, `clear_manual` and
`status`; the loop thread calls `tick`.
"""

from __future__ import annotations

import logging
import threading

from . import buildinfo
from .device import DeviceLink
from .mapping import END, Event
from .sessions import Sessions, State

MANUAL = ("manual", "manual")  # agent, session of the manual state (BRG-12)

log = logging.getLogger(__name__)


class Bridge:
    def __init__(self, sessions: Sessions, link: DeviceLink) -> None:
        self.sessions = sessions
        self.link = link
        self._lock = threading.Lock()

    def _apply(self, now: float, cause: str) -> None:
        self.sessions.expire(now)
        merged = self.sessions.merged()
        if merged != self.link.state:
            delivery = "sent" if self.link.connected else "device not connected"
            log.info("state %s (%s) <- %s", merged.value, delivery, cause)
        self.link.set_state(merged, now)

    def event(self, event: Event, now: float) -> None:
        with self._lock:
            cause = f"{event.agent} {event.session} {event.name}"
            log.debug("event %s -> %s", cause, getattr(event.state, "value", event.state))
            if event.state is None:
                return
            if event.state == END:
                self.sessions.end(event.agent, event.session)
            else:
                assert isinstance(event.state, State)
                self.sessions.update(event.agent, event.session, event.state, now)
            self._apply(now, cause)

    def set_manual(self, state: State, now: float) -> None:
        with self._lock:
            self.sessions.update(*MANUAL, state, now)
            self._apply(now, f"manual {state.value}")

    def clear_manual(self, now: float) -> None:
        with self._lock:
            self.sessions.end(*MANUAL)
            self._apply(now, "manual clear")

    def tick(self, now: float) -> None:
        with self._lock:
            self._apply(now, "session timeout")
            self.link.step(now)

    def status(self, now: float) -> dict:
        with self._lock:
            version, commit = buildinfo.version(), buildinfo.commit()
            return {
                "bridge": {"version": version, "commit": commit},
                "device": {
                    "connected": self.link.connected,
                    "port": self.link.device,
                    "firmware": self.link.firmware,
                },
                "state": self.sessions.merged().value,
                "sessions": [
                    {
                        "agent": s.agent,
                        "session": s.session,
                        "state": s.state.value,
                        "idle_s": round(now - s.last_event),
                    }
                    for s in self.sessions.all()
                ],
            }


def status_text(status: dict) -> str:
    """Human-readable status for `cody status` (BRG-11)."""
    device = status["device"]
    if device["connected"]:
        line = f"device:   connected on {device['port']}, firmware {device['firmware']}"
    else:
        line = "device:   not connected"
    lines = [
        f"bridge:   {status['bridge']['version']} {status['bridge']['commit']}",
        line,
        f"state:    {status['state']}",
        f"sessions: {len(status['sessions'])}",
    ]
    lines += [
        f"  {s['agent']:<12} {s['state']:<10} {s['session']}  (last event {s['idle_s']} s ago)"
        for s in status["sessions"]
    ]
    return "\n".join(lines) + "\n"
