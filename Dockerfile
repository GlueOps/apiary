# The runtime image. It compiles nothing.
#
# Both binaries are cross-compiled by goreleaser before this runs and land in the
# build context ready to copy — see the dockers_v2 block in .goreleaser.yaml.
# That is the point of the split: this project used to describe how it builds in
# two places, and they had already drifted. The Dockerfile stamped
# `main.version`, which does not exist in the control plane, and `go build -X`
# on an absent symbol is silently ignored — so every image reported itself as
# "dev / none / unknown" however carefully the build args were set, while the
# goreleaser archives reported correctly. `apiary version` is what you read off
# a running server to find out what is deployed.
#
# Build it with `goreleaser release --snapshot --clean`, not `docker build .`;
# there is no source here to compile.
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# Set by buildx, and it is how the two platform builds tell themselves apart:
# goreleaser stages each platform's binaries under linux/amd64/ and linux/arm64/
# in the build context, then runs one `buildx build --platform ...` per platform.
# Hard-coding a path here would silently put the amd64 binary in the arm64 image.
ARG TARGETPLATFORM

# ca-certificates for outbound TLS (Proxmox APIs, SMTP, OIDC); tzdata because
# schedules and audit timestamps are rendered in a configured zone rather than
# UTC-only, and Alpine ships no zone database.
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S apiary \
    && adduser -S -G apiary -H -s /sbin/nologin apiary

COPY ${TARGETPLATFORM}/apiary /usr/local/bin/apiary

# The agent the onboarding job installs onto a hypervisor over SSH.
#
# A FILE, not embedded in the control-plane binary. It could go into
# internal/api/webdist and be picked up by `//go:embed all:webdist`, but that
# directory is served by the SPA's static handler — so the binary would be
# downloadable by anyone who can reach the frontend — and vite runs with
# `emptyOutDir: true`, so the next local `yarn build` would delete it. Kept
# separate, replacing the agent does not mean rebuilding and redeploying the
# control plane.

USER apiary
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["apiary"]
CMD ["serve"]
