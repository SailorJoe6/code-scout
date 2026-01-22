# Developer Container (Docker, Podman, Apple Container)

This guide sets up a containerized dev environment with Go, the codex and
Claude CLIs, and the build tooling needed for Code Scout.

## What's Included

- Go 1.25.x toolchain (from the base image)
- Node.js + `codex`, `claude`, and `bd` (beads) CLIs
- Build tools for CGO (`build-essential`, `pkg-config`, `libssl-dev`)
- Common utilities (`curl`, `git`, `ripgrep`, `jq`)
- Passwordless `sudo` for the dev user

## Build And Run (Docker)

```bash
# Linux: preserve file ownership on the bind mount
export LOCAL_UID=$(id -u)
export LOCAL_GID=$(id -g)

docker compose -f docker-compose.dev.yml build
docker compose -f docker-compose.dev.yml run --rm dev
```

The repository is bind-mounted at `/workspaces/code_scout` with read-write
access. Go build caches and HuggingFace caches are stored in named volumes.

## Build And Run (Podman)

If `podman compose` routes through `docker-compose` and fails to attach
(`unable to upgrade to tcp, received 409`), switch to the podman-compose
provider:

```bash
PODMAN_COMPOSE_PROVIDER=podman-compose podman compose -f docker-compose.dev.yml build
PODMAN_COMPOSE_PROVIDER=podman-compose podman compose -f docker-compose.dev.yml run --rm dev
```

Direct podman run (bypasses compose and avoids remote pulls):

```bash
podman build -t localhost/code-scout-dev -f Dockerfile.dev .
podman run --rm -it \
  -v "$PWD:/workspaces/code_scout" \
  -v code_scout_go-mod:/home/dev/go/pkg/mod \
  -v code_scout_go-build:/home/dev/.cache/go-build \
  -v code_scout_hf-cache:/home/dev/.cache/huggingface \
  -w /workspaces/code_scout \
  --pull=never \
  localhost/code-scout-dev
```

## Build And Run (Apple `container`)

Apple's `container` CLI does not currently support Compose. Use build/run:

```bash
container system start
container build -t code-scout-dev -f Dockerfile.dev .
container run --rm -it \
  -v "$PWD:/workspaces/code_scout" \
  -w /workspaces/code_scout \
  code-scout-dev
```

## First-Time Repo Setup (Inside The Container)

Download the LanceDB native artifacts into the repo:

```bash
curl -sSL https://raw.githubusercontent.com/lancedb/lancedb-go/main/scripts/download-artifacts.sh | bash
```

`./fix-dylib-paths.sh` is macOS-only and does not apply in the Linux container.

## Building Linux Bundles (Inside The Container)

Use the container to produce Linux bundles on any host:

```bash
TARGETS=linux_amd64 ./build.sh
```

The output bundles land in `dist/` on the host because the repo is bind-mounted.

## Codex, Claude, And Beads

`codex`, `claude`, and `bd` are installed. Set credentials before using them:

```bash
export OPENAI_API_KEY=...  # or your preferred codex auth env var
codex --help

claude --help

bd --help
```

You can also run the existing wrapper script with `./ralph/start --codex` (host) or
use the container workflow below.
The container installs `claude` via npm (upstream npm install is deprecated,
but it is the most reliable non-interactive option for containers).
Beads is installed from npm (`@beads/bd`); run `bd prime` for workflow context.

### Running ralph in the dev container

Start the container once in the background:

```bash
podman run -d --name code-scout-dev \
  -v "$PWD:/workspaces/code_scout" \
  -v code_scout_go-mod:/home/dev/go/pkg/mod \
  -v code_scout_go-build:/home/dev/.cache/go-build \
  -v code_scout_hf-cache:/home/dev/.cache/huggingface \
  -w /workspaces/code_scout \
  --pull=never \
  localhost/code-scout-dev sleep infinity
```

Authenticate inside the container (set API keys, etc.):

```bash
podman exec -it code-scout-dev /bin/bash
```

Run the wrapper from the host. It uses `podman exec`, so exit codes propagate:

```bash
./ralph/start --container code-scout-dev --codex
# or for Claude:
./ralph/start --container code-scout-dev
```

When finished, stop and remove the container:

```bash
podman stop code-scout-dev
podman rm code-scout-dev
```

## TEI And Embeddings

The dev image does not include TEI (`text-embeddings-router`) by default.
Use a host TEI instance, or install TEI inside the container if needed.

To install TEI inside the container, follow the build-from-source steps in
[TEI_SETUP.md](TEI_SETUP.md) (requires Rust). After installation:

```bash
text-embeddings-router --version
```

You can then run TEI inside the container:

```bash
text-embeddings-router \
  --model-id nomic-ai/nomic-embed-text-v1.5 \
  --port 8080
```

If you prefer to run TEI on the host (GPU, Metal, etc.), point
`.code-scout.json` at your host endpoint:

- Docker Desktop: `http://host.docker.internal:8080`
- Podman: `http://host.containers.internal:8080`

## Notes

- This container is for development; it does not run TEI or the wrapper by
  default.
- The image is intentionally minimal; install additional tools via `sudo apt`.
