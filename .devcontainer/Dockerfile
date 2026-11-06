# Development environment and CI environment (spec/delivery.md 2).
# Every tool has a pinned version (spec/architecture.md 5.1).
FROM mcr.microsoft.com/devcontainers/base:bookworm@sha256:99f5250dd2d5f8b11046d5662dd94c1e3c1dae56b560c7caad5ceb7edf349046

# Mount point for the shared claude-config volume, owned by vscode so the empty
# volume inherits that ownership on first mount
RUN mkdir -p /home/vscode/.claude && chown vscode:vscode /home/vscode/.claude

# Host compiler for the firmware unit tests (PlatformIO native environment);
# libgl1 for OpenCascade (build123d)
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential xz-utils libgl1 \
    && rm -rf /var/lib/apt/lists/*

# shellcheck, checksum-verified
ARG SHELLCHECK_VERSION=0.11.0
ARG SHELLCHECK_SHA256_X86_64=8c3be12b05d5c177a04c29e3c78ce89ac86f1595681cab149b65b97c4e227198
ARG SHELLCHECK_SHA256_AARCH64=12b331c1d2db6b9eb13cfca64306b1b157a86eb69db83023e261eaa7e7c14588
RUN arch="$(uname -m)" \
    && case "$arch" in \
         x86_64) sha="$SHELLCHECK_SHA256_X86_64" ;; \
         aarch64) sha="$SHELLCHECK_SHA256_AARCH64" ;; \
         *) echo "unsupported architecture $arch" >&2; exit 1 ;; \
       esac \
    && curl -fsSL -o /tmp/shellcheck.tar.xz \
         "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.linux.${arch}.tar.xz" \
    && echo "${sha}  /tmp/shellcheck.tar.xz" | sha256sum -c - \
    && tar -xJf /tmp/shellcheck.tar.xz -C /tmp \
    && install -m 0755 "/tmp/shellcheck-v${SHELLCHECK_VERSION}/shellcheck" /usr/local/bin/shellcheck \
    && rm -rf /tmp/shellcheck*

# uv (pinned image tag) manages Python, the workspace environment and Python tools
COPY --from=ghcr.io/astral-sh/uv:0.13.7 /uv /uvx /usr/local/bin/

# Shared locations. devcontainers/ci changes the vscode user's UID to the
# runner's, so everything that is written at runtime (uv cache, workspace
# environment, PlatformIO core dir) must be writable for every user. Each step
# ends with a chmod in its own layer (a chmod in a later layer would copy the
# files) and removes its download caches, which the image doesn't need.
ENV UV_PYTHON_INSTALL_DIR=/opt/uv/python \
    UV_TOOL_DIR=/opt/uv/tools \
    UV_TOOL_BIN_DIR=/usr/local/bin \
    UV_PROJECT_ENVIRONMENT=/opt/cody-venv \
    UV_LINK_MODE=copy \
    UV_CACHE_DIR=/opt/uv/cache \
    PLATFORMIO_CORE_DIR=/opt/platformio
RUN mkdir -p /opt/uv /opt/platformio /opt/cody-venv \
    && chown vscode:vscode /opt/platformio /opt/cody-venv \
    && chmod 0777 /opt/uv /opt/platformio /opt/cody-venv

# Python, PlatformIO and clang-format as uv tools
COPY .python-version /tmp/cody/.python-version
RUN uv python install "$(cat /tmp/cody/.python-version)" \
    && uv tool install --python "$(cat /tmp/cody/.python-version)" platformio==6.2.0 \
    && uv tool install clang-format==23.1.5 \
    && uv cache clean \
    && chmod -R a+rwX /opt/uv

USER vscode

# ESP32 toolchain (pioarduino platform, compiler, Arduino core, native platform),
# installed from the firmware's platformio.ini so the pins live in one place.
# pioarduino installs part of its tools (esptool, Python env) only on the first
# build, so an empty sketch is built once here; later builds work offline.
# Slimming (about 5 GB): download archives and caches are removed; the Arduino
# libraries are kept only for the ESP32-C3 (TD-01; another chip needs them
# back); pioarduino's second copy of the RISC-V compiler and debugger in tools/
# is replaced by a link to the identical copy in packages/.
COPY --chown=vscode:vscode firmware/platformio.ini /tmp/cody/firmware/platformio.ini
RUN cd /tmp/cody/firmware \
    && pio pkg install --environment esp32c3 \
    && pio pkg install --environment native \
    && mkdir -p src \
    && printf '#include <Arduino.h>\nvoid setup() {}\nvoid loop() {}\n' > src/main.cpp \
    && pio run --environment esp32c3 \
    && rm -rf /opt/platformio/dist /opt/platformio/.cache \
    && find /opt/platformio/packages/framework-arduinoespressif32-libs -mindepth 1 -maxdepth 1 \
         -type d ! -name esp32c3 ! -name hosted -exec rm -rf {} + \
    && for t in toolchain-riscv32-esp tool-riscv32-esp-elf-gdb; do \
         rm -rf "/opt/platformio/tools/$t" && ln -s "../packages/$t" "/opt/platformio/tools/$t"; \
       done \
    && chmod -R a+rwX /opt/platformio

# Python dependencies of the workspace from the lockfile (workspace members
# themselves are installed by `uv sync` in the checkout, see postCreateCommand)
COPY --chown=vscode:vscode pyproject.toml uv.lock .python-version /tmp/cody/
COPY --chown=vscode:vscode bridge/pyproject.toml /tmp/cody/bridge/pyproject.toml
COPY --chown=vscode:vscode cad/pyproject.toml /tmp/cody/cad/pyproject.toml
COPY --chown=vscode:vscode tools/pyproject.toml /tmp/cody/tools/pyproject.toml
RUN cd /tmp/cody \
    && uv sync --frozen --no-install-workspace \
    && uv cache clean \
    && rm -rf /opt/uv/cache \
    && chmod -R a+rwX /opt/cody-venv

USER root
RUN rm -rf /tmp/cody
