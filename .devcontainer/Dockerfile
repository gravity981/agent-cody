# Development environment; by default also the CI environment (spec/delivery.md 2).
# TODO(blueprint): choose the base image for the project's toolchain; any image
# with a "vscode" user works (mcr.microsoft.com/devcontainers/*).
FROM mcr.microsoft.com/devcontainers/base:bookworm

# Mount point for the shared claude-config volume, owned by vscode so the empty
# volume inherits that ownership on first mount
RUN mkdir -p /home/vscode/.claude && chown vscode:vscode /home/vscode/.claude

# TODO(blueprint): the project's toolchain (compilers, SDKs, CAD tools, test
# runners), each with a pinned version and, where downloaded, a checksum check.
# Keep it in this file, not in postCreateCommand, so CI gets the same tools.
#
# RUN apt-get update \
#     && apt-get install -y --no-install-recommends <packages> \
#     && rm -rf /var/lib/apt/lists/*
