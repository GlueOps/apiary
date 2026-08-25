# syntax=docker/dockerfile:1

# ---- Stage 1: the single-page app -------------------------------------------
#
# corepack, not the bundled yarn. The node image ships Yarn 1.22, the repository
# declares `packageManager: yarn@4.18.0`, and Yarn 1 cannot read a Yarn 4
# lockfile — it reports "Your lockfile needs to be updated", which reads like a
# stale lockfile rather than the wrong yarn entirely.
FROM node:24.19.0-alpine AS frontend
WORKDIR /app
RUN corepack enable

# .yarnrc.yml carries `nodeLinker: node-modules`. Without it Yarn 4 falls back to
# Plug'n'Play inside the image and resolves differently from every developer
# machine; package.json is what corepack reads to pick the yarn version.
COPY package.json .yarnrc.yml ./
COPY ui/package.json ui/yarn.lock ui/

WORKDIR /app/ui
RUN --mount=type=cache,target=/root/.yarn/berry/cache yarn install --immutable

COPY ui/ ./
# vite's outDir is `../internal/api/webdist`, so the build lands at
# /app/internal/api/webdist — NOT /app/ui/dist, which never exists.
RUN yarn build

# ---- Stage 2: both Go binaries ----------------------------------------------
#
# One stage, not two. All four modules are joined by go.work and by replace
# directives pointing at sibling directories, so a stage holding only ./agent
# cannot build it: `replacement directory ../sdk/go does not exist`. Once the
# tree is present for one binary it is present for both, and a second stage
# would re-download the same module graph to produce the second.
FROM golang:1.27-alpine AS build
WORKDIR /src

# The module graph before the source, so editing a .go file does not re-resolve
# dependencies. Every module named by go.work must be here, including the ones
# reached only through a replace directive — `go mod download` fails outright
# when a replacement directory is missing.
COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY pkg/proxmox/go.mod pkg/proxmox/go.sum pkg/proxmox/
COPY sdk/go/go.mod sdk/go/go.sum sdk/go/
COPY agent/go.mod agent/go.sum agent/
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# After `COPY . .`, so it is not overwritten by whatever the context happens to
# hold. .dockerignore excludes internal/api/webdist for the same reason: a
# developer's local build output must not decide what ships.
COPY --from=frontend /app/internal/api/webdist/ internal/api/webdist/

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

# internal/buildinfo, NOT main. There is no `main.version` in this repository, and
# `go build -X` on a symbol that does not exist is silently ignored — so the
# previous flags compiled fine and every image reported "dev / none / unknown"
# however carefully the build args were set. `apiary version` is what you read
# off a running server to find out what is deployed, so it was wrong in the one
# place it matters. These paths must stay in step with .goreleaser.yaml, which
# sets the same three variables; they are the same build expressed twice.

# -trimpath so the binary does not record /src and the build is reproducible.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w \
      -X github.com/glueops/apiary/internal/buildinfo.Version=${VERSION} \
      -X github.com/glueops/apiary/internal/buildinfo.Commit=${COMMIT} \
      -X github.com/glueops/apiary/internal/buildinfo.Date=${BUILD_DATE}" \
    -o /out/apiary .

# main.*, unlike the server above. The agent is a separate module
# (github.com/glueops/apiary-agent) and stamps agent/main.go's package-level
# vars, which it then copies into agent.Version. Using the server's buildinfo
# path here would be another silently-ignored -X.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd agent && CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w \
      -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
    -o /out/apiary-agent .

# ---- Stage 3: what actually ships -------------------------------------------
FROM alpine:3.24
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S apiary && adduser -S -G apiary -H -s /sbin/nologin apiary

COPY --from=build /out/apiary /usr/local/bin/apiary

# The agent binary the onboarding job installs onto a hypervisor over SSH.
#
# NOT under internal/api/webdist. That directory is `//go:embed all:webdist` and
# is served by the SPA's static handler, so a binary placed there is downloadable
# by anyone who can reach the frontend — and vite is configured with
# `emptyOutDir: true`, so the next local `yarn build` deletes it. Kept as a file
# here rather than embedded so replacing the agent does not mean rebuilding and
# redeploying the control plane.
COPY --from=build /out/apiary-agent /usr/local/share/apiary/apiary-agent

USER apiary
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["apiary"]
CMD ["serve"]
