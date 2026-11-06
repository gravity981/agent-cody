"""cody-bridge: runs the bridge (spec/architecture.md 4.1, 4.4); `--version` (REL-02, BRG-10)."""

from __future__ import annotations

import argparse
import logging
import os
import sys
import time
from pathlib import Path

from . import buildinfo
from .bridge import Bridge
from .config import ConfigError, load
from .device import DeviceLink, pyserial_lister, pyserial_opener
from .server import serve
from .sessions import Sessions

DEFAULT_CONFIG = Path(os.environ.get("CODY_CONFIG", "~/.config/cody/bridge.toml")).expanduser()
LOOP_S = 0.1


def version() -> str:
    return buildinfo.version()


def commit() -> str:
    return buildinfo.commit()


LOG_FORMAT = "%(asctime)s %(levelname)-5s %(message)s"
LOG_DATE = "%Y-%m-%d %H:%M:%S"


def setup_logging(level: str) -> None:
    """Formatted lines on stdout (BRG-13); Docker keeps and rotates them (BRG-14)."""
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(logging.Formatter(LOG_FORMAT, LOG_DATE))
    root = logging.getLogger()
    root.handlers[:] = [handler]
    root.setLevel(logging.DEBUG if level == "debug" else logging.INFO)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="cody-bridge", description="Cody host bridge")
    parser.add_argument(
        "--version", action="store_true", help="print version and commit, then exit"
    )
    parser.add_argument("--config", type=Path, default=DEFAULT_CONFIG, help="config file (TOML)")
    args = parser.parse_args(argv)
    if args.version:
        print(f"cody-bridge {version()} {commit()}")
        return 0

    try:
        config = load(args.config)
    except ConfigError as exc:
        print(f"cody-bridge: config: {exc}", file=sys.stderr)
        return 2
    setup_logging(config.log_level)
    logging.info("cody-bridge %s %s, config %s", version(), commit(), args.config)

    link = DeviceLink(
        pyserial_lister, pyserial_opener, config.speed, config.volume, config.serial_port
    )
    bridge = Bridge(Sessions(config.session_timeout_min * 60), link)
    if not serve(bridge, config.listen, config.port):
        print("cody-bridge: no address to listen on", file=sys.stderr)
        return 1
    try:
        while True:
            bridge.tick(time.monotonic())
            time.sleep(LOOP_S)
    except KeyboardInterrupt:
        return 0


if __name__ == "__main__":
    sys.exit(main())
