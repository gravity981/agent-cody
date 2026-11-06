"""cody-bridge command line. Skeleton from step 1c: reports version and commit (REL-02).

The bridge itself (BRG) follows in stage 2.
"""

from __future__ import annotations

import argparse
import os
import sys


def version() -> str:
    """Version of this build; set in the image from the build (spec/architecture.md 6)."""
    return os.environ.get("CODY_VERSION") or "0.0.0+unknown"


def commit() -> str:
    return os.environ.get("CODY_COMMIT") or "unknown"


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="cody-bridge", description="Cody host bridge")
    parser.add_argument(
        "--version", action="store_true", help="print version and commit, then exit"
    )
    args = parser.parse_args(argv)
    if args.version:
        print(f"cody-bridge {version()} {commit()}")
        return 0
    parser.print_usage(sys.stderr)
    print("cody-bridge: the bridge is not implemented yet (stage 2)", file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main())
