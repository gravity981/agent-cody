"""HTTP endpoints (BRG-01, BRG-08, BRG-11, BRG-12; spec/architecture.md 4.1)."""

from __future__ import annotations

import json
import logging
import threading
import time
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

from .bridge import Bridge, status_text
from .mapping import AGENTS, PayloadError
from .sessions import State

log = logging.getLogger(__name__)
MAX_BODY = 1_000_000


def make_handler(bridge: Bridge, clock=time.monotonic) -> type[BaseHTTPRequestHandler]:
    class Handler(BaseHTTPRequestHandler):
        server_version = "cody-bridge"

        def log_message(self, format: str, *args: object) -> None:  # noqa: A002
            log.debug("%s %s", self.address_string(), format % args)

        def _reply(self, status: HTTPStatus, body: str, content_type: str = "text/plain") -> None:
            data = body.encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", f"{content_type}; charset=utf-8")
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)

        def _json_body(self) -> dict | None:
            length = int(self.headers.get("Content-Length") or 0)
            if length <= 0 or length > MAX_BODY:
                return None
            try:
                body = json.loads(self.rfile.read(length))
            except json.JSONDecodeError, UnicodeDecodeError:
                return None
            return body if isinstance(body, dict) else None

        def do_GET(self) -> None:  # noqa: N802
            url = urlparse(self.path)
            if url.path != "/status":
                self._reply(HTTPStatus.NOT_FOUND, "not found\n")
                return
            status = bridge.status(clock())
            if parse_qs(url.query).get("format") == ["text"]:
                self._reply(HTTPStatus.OK, status_text(status))
            else:
                self._reply(HTTPStatus.OK, json.dumps(status) + "\n", "application/json")

        def do_POST(self) -> None:  # noqa: N802
            path = urlparse(self.path).path
            if path.startswith("/hooks/"):
                mapper = AGENTS.get(path[len("/hooks/") :])
                if mapper is None:
                    self._reply(HTTPStatus.NOT_FOUND, "unknown agent\n")
                    return
                body = self._json_body()
                if body is None:
                    self._reply(HTTPStatus.BAD_REQUEST, "expected a JSON object\n")
                    return
                try:
                    event = mapper(body)
                except PayloadError as exc:
                    self._reply(HTTPStatus.BAD_REQUEST, f"{exc}\n")
                    return
                bridge.event(event, clock())
                self._reply(HTTPStatus.OK, "ok\n")
            elif path == "/state":
                body = self._json_body()
                try:
                    state = State(body.get("state")) if body else None
                except ValueError:
                    state = None
                if state is None:
                    names = ", ".join(s.value for s in State)
                    self._reply(HTTPStatus.BAD_REQUEST, f"state must be one of {names}\n")
                    return
                bridge.set_manual(state, clock())
                self._reply(HTTPStatus.OK, "ok\n")
            else:
                self._reply(HTTPStatus.NOT_FOUND, "not found\n")

        def do_DELETE(self) -> None:  # noqa: N802
            if urlparse(self.path).path != "/state":
                self._reply(HTTPStatus.NOT_FOUND, "not found\n")
                return
            bridge.clear_manual(clock())
            self._reply(HTTPStatus.OK, "ok\n")

    return Handler


def serve(bridge: Bridge, addresses: list[str], port: int) -> list[ThreadingHTTPServer]:
    """Listens on exactly these addresses (BRG-08); an address that doesn't
    exist on this host (e.g. no Docker) is skipped with a warning."""
    handler = make_handler(bridge)
    servers = []
    for address in addresses:
        try:
            server = ThreadingHTTPServer((address, port), handler)
        except OSError as exc:
            log.warning("can't listen on %s:%d: %s", address, port, exc)
            continue
        server.daemon_threads = True
        threading.Thread(target=server.serve_forever, daemon=True).start()
        log.info("listening on %s:%d", address, port)
        servers.append(server)
    return servers
