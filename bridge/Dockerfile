# Bridge image (spec/architecture.md TD-04), built and pushed by ci/publish.
# Build context: the repository root (uv workspace and lockfile).
FROM python:3.14.10-slim-bookworm@sha256:3fe1a0b6c2d6b2ebb572906cdabeca75aeb558c0c17db5ffefbcf8094871c32b

COPY --from=ghcr.io/astral-sh/uv:0.13.7 /uv /usr/local/bin/uv

ENV UV_PROJECT_ENVIRONMENT=/opt/venv \
    UV_LINK_MODE=copy \
    UV_COMPILE_BYTECODE=1 \
    UV_PYTHON_DOWNLOADS=never \
    UV_NO_CACHE=1

WORKDIR /src
# The workspace needs every member's pyproject.toml; only the bridge is installed
COPY pyproject.toml uv.lock .python-version ./
COPY cad/pyproject.toml cad/pyproject.toml
COPY tools/pyproject.toml tools/pyproject.toml
COPY bridge bridge
COPY LICENSE /usr/share/doc/cody-bridge/LICENSE
RUN uv sync --frozen --no-dev --no-editable --package cody-bridge \
    && rm -rf /src

ARG VERSION=0.0.0+unknown
ARG COMMIT=unknown
ENV CODY_VERSION=${VERSION} \
    CODY_COMMIT=${COMMIT}

USER nobody
ENTRYPOINT ["/opt/venv/bin/cody-bridge"]
