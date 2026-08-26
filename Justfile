generate name:
  go run . generate migrations {{name}}

sdk:
  go run . generate sdk

# Cross-compile the agent into the control plane's distribution directory.
#
# A hypervisor in a managed estate has no route to the internet, so the control
# plane is the only host it can fetch a new agent from. These binaries are
# embedded at compile time, which means this has to run BEFORE `go build` of the
# server — and that ordering is why it is its own recipe rather than a hook.
#
# linux only: the agent is a systemd service that runs as root on a hypervisor,
# and there is no darwin host for it to run on.
agent-dist:
  #!/usr/bin/env bash
  set -euo pipefail
  for arch in amd64 arm64; do
    echo "  building linux/$arch"
    ( cd agent && GOWORK=off GOOS=linux GOARCH="$arch" CGO_ENABLED=0 \
        go build -trimpath -ldflags "-s -w" \
        -o "../internal/agentdist/dist/apiary-agent_linux_$arch" . )
  done
  ls -lh internal/agentdist/dist/ | tail -n +2

release:
    goreleaser build --snapshot --clean

up:
  docker-compose up -d
  just migrate up

# Prints the test database DSN, honouring TEST_DATABASE_URL if it is already set.
#
# Derived from .env rather than written out here, so the password lives in one
# place. This is a shebang recipe, which just does not echo — the DSN must not
# land in terminal scrollback.
#
# Note the database is apiary_test on the DEVELOPMENT server, not a schema inside
# the application database. Tests create and drop schemas wholesale; sharing
# $DB_NAME means the suite migrates and truncates real data.
[private]
_test-dsn:
  #!/usr/bin/env bash
  set -euo pipefail
  if [ -n "${TEST_DATABASE_URL:-}" ]; then echo "$TEST_DATABASE_URL"; exit 0; fi
  set -a; . ./.env 2>/dev/null || true; set +a
  echo "postgres://${DB_USER}:${DB_PASSWORD}@localhost:${LOCAL_POSTGRES_PORT}/apiary_test?sslmode=disable"

# postgres/initdb/10-create-extra-db.sh only runs on a FRESH data directory, so
# anyone whose volume predates apiary_test needs this. Idempotent, and it also
# covers atlas_dev — the init script's comment promises a `just atlas-dev` that
# has never existed.
# Create the scratch databases on a running server.
test-db:
  #!/usr/bin/env bash
  set -euo pipefail
  set -a; . ./.env 2>/dev/null || true; set +a
  admin="postgres://${DB_USER}:${DB_PASSWORD}@localhost:${LOCAL_POSTGRES_PORT}/${DB_NAME}?sslmode=disable"
  for db in atlas_dev apiary_test; do
    if [ "$db" = "${DB_NAME}" ]; then
      echo "refusing to treat the application database as scratch" >&2; exit 1
    fi
    if psql "$admin" -tAc "SELECT 1 FROM pg_database WHERE datname='$db'" | grep -q 1; then
      echo "database '$db' already exists"
    else
      psql "$admin" -qc "CREATE DATABASE \"$db\" OWNER \"${DB_USER}\""
      echo "created database '$db'"
    fi
  done

# `go test` SKIPS every database-backed test when TEST_DATABASE_URL is unset, and
# a run that skipped half its assertions is indistinguishable from one that
# passed. Fail instead.
[private]
_require-test-db:
  #!/usr/bin/env bash
  set -euo pipefail
  command -v psql >/dev/null 2>&1 || { echo "note: psql not found, skipping pre-flight" >&2; exit 0; }
  dsn="$(just _test-dsn)"
  psql "$dsn" -tAc 'select 1' >/dev/null 2>&1 || {
    echo "test database unreachable — run:  just test-db" >&2; exit 1; }

test: _require-test-db
  #!/usr/bin/env bash
  set -euo pipefail
  TEST_DATABASE_URL="$(just _test-dsn)" go test ./... -race
  cd ui && yarn tsc --noEmit && yarn lint && yarn format:check && yarn test

# The per-function list is the actionable half; the total is the number people
# quote. Needs the test database or the figures are fiction.
# Coverage, worst-first.
cover: _require-test-db
  #!/usr/bin/env bash
  set -euo pipefail
  export TEST_DATABASE_URL="$(just _test-dsn)"
  go test ./... -coverprofile=coverage.out -covermode=atomic >/dev/null
  echo "── per package ─────────────────────────────────────────────"
  go test ./... -cover 2>/dev/null | grep -E 'coverage:' \
    | sed -E 's#github.com/glueops/apiary/?##' \
    | awk '{for(i=1;i<=NF;i++) if($i=="coverage:") print $(i+1), $0}' \
    | sort -n | cut -d' ' -f2- || true
  echo "── least covered functions ─────────────────────────────────"
  go tool cover -func=coverage.out | grep -v '100.0%$' | sort -k3 -n | head -20
  echo "── total ───────────────────────────────────────────────────"
  go tool cover -func=coverage.out | tail -1

# Coverage as a browsable HTML report.
cover-html: cover
  go tool cover -html=coverage.out -o coverage.html
  @echo "wrote coverage.html"

# CRAP weighs complexity against coverage, so it ranks what to test NEXT — which
# raw coverage cannot: a 0%-covered one-liner and a 0%-covered 30-branch function
# are the same number to `go tool cover` and very different risks. Needs the test
# database, or every DB-backed function reads as uncovered and the rank is noise.
# Risk ranking: complexity vs coverage.
crap: _require-test-db
  #!/usr/bin/env bash
  set -euo pipefail
  TEST_DATABASE_URL="$(just _test-dsn)" go-crap scan

# Everything a reviewer would want before approving.
quality: test crap
  golangci-lint run

down *args:
  docker-compose down {{args}}

reset:
  down -v
  up

terraform:
  go run . generate terraform openapi-generator

migrate *args:
  go run . migrate {{args}}

readme:
  npx repomix --include "CLAUDE.md,AGENTS.md,Justfile,Dockerfile,docker-compose*.yml,go.mod,agent/go.mod,ui/package.json,.goreleaser.yaml,cmd/**/*.go,internal/handlers/*.go,internal/models/**/*.go,ui/src/routes/index.tsx,mprocs.yaml" -o repomix-readme.md
  cat repomix-readme.md | claude --dangerously-skip-permissions -p "You are a technical writer. Using the codebase snapshot from stdin, generate a comprehensive README.md for this project. Include: project overview, features list, tech stack, prerequisites, setup/installation, development workflow, architecture overview, API documentation summary (reference /docs for full OpenAPI spec), deployment notes, and contributing guidelines. Keep it professional and concise. Do NOT include the repomix snapshot in the output — just the README content." --model haiku > README.md
  rm -f repomix-readme.md
  @echo "README.md generated successfully."

setup-pgbouncer:
  go run . setup pgbouncer

setup-postgres:
  go run . setup postgres

seed:
  go run . seed

# Scan the committed history for secrets.
#
# `gitleaks git`, not `gitleaks dir`, and the difference matters: dir scans the
# working tree including files git ignores, so it reports .env — which holds real
# credentials and is supposed to. What this must answer is "has anything secret
# left this machine", and that is exactly what git mode looks at.
#
# .env is deliberately NOT allowlisted in .gitleaks.toml, so if it is ever
# force-added past the ignore rule this trips immediately.
[doc("Scan the committed history for secrets")]
secrets:
  gitleaks git . --config .gitleaks.toml --redact --verbose

# Scan what is staged, before it becomes a commit.
#
# The cheap one to run by hand before a push, and what a pre-commit hook would
# call. Nothing installs a hook here on purpose: core.hooksPath is commonly set
# globally, so writing one would either be ignored or apply to every repository
# on the machine.
[doc("Scan staged changes for secrets, before they become a commit")]
secrets-staged:
  gitleaks git . --config .gitleaks.toml --redact --verbose --staged


# ---- Lab: fleet operations against real clusters ----------------------------

# Loads .env.lab and fails loudly if anything it needs is missing.
#
# A dependency of the lab recipes rather than something each repeats, so a
# half-filled .env.lab is one clear message instead of a curl that 403s or a test
# that skips with "APIARY_CYCLE_* is not set" and looks like it passed.
[private]
_lab-env:
  #!/usr/bin/env bash
  set -euo pipefail
  [ -f .env.lab ] || { echo "no .env.lab — copy .env.lab.example and fill it in" >&2; exit 1; }
  set -a; . ./.env.lab; set +a
  missing=""
  for k in LAB_A_API LAB_A_TOKEN_ID LAB_A_SECRET LAB_A_PEER \
           LAB_B_API LAB_B_TOKEN_ID LAB_B_SECRET LAB_B_PEER \
           LAB_MOVING LAB_LINK; do
    [ -n "${!k:-}" ] || missing="$missing $k"
  done
  [ -z "$missing" ] || { echo "unset in .env.lab:$missing" >&2; exit 1; }

# Which cluster holds which node, right now.
#
# Worth running either side of a move: it is the before-and-after of the demo,
# and it reads the API rather than SSH because the API view is what placement
# acts on.
[doc("Show which lab cluster currently holds which node")]
lab-topology: _lab-env
  #!/usr/bin/env bash
  set -euo pipefail
  set -a; . ./.env.lab; set +a
  python3 scripts/lab/topology.py "cluster A" "$LAB_A_API" "$LAB_A_TOKEN_ID" "$LAB_A_SECRET"
  python3 scripts/lab/topology.py "cluster B" "$LAB_B_API" "$LAB_B_TOKEN_ID" "$LAB_B_SECRET"

# Move a hypervisor between the two lab clusters, end to end.
#
#   just fleet-move            A -> B
#   just fleet-move reverse    B -> A
#
# DESTRUCTIVE, AND IT DOES NOT PUT THE NODE BACK. It drives the real thing:
# capacity sync, the departure gate refusing twice, cordon, delnode on the
# source, join on the destination, and a re-sync of both. A failure part-way can
# leave the node standalone — in neither cluster — which is recoverable, but is a
# state a live demo should not meet for the first time.
#
# The node named by LAB_MOVING must have NO guests. The gate refuses one that is
# still carrying any, which is the gate working rather than the demo breaking.
#
# A shebang recipe so `just` does not echo it: the token secrets reach the test
# through this environment and must not land in terminal scrollback.
[doc("Move a hypervisor between the lab clusters (direction: forward|reverse)")]
fleet-move direction="forward": _lab-env _require-test-db
  #!/usr/bin/env bash
  set -euo pipefail
  set -a; . ./.env.lab; set +a
  set -a; . ./.env; set +a
  export TEST_DATABASE_URL="$(just _test-dsn)"
  # goreleaser and go both trip over a stale GOROOT left by a removed toolchain;
  # the Makefile unexports it for the same reason.
  unset GOROOT
  # Direction is only ever a swap of A and B — the node that moves and the link
  # it advertises do not change. Sending a node back is therefore one word rather
  # than an edit of .env.lab, which is exactly the edit a live demo fumbles.
  case "{{direction}}" in
    forward) from=A; to=B; s=A; d=B ;;
    reverse) from=B; to=A; s=B; d=A ;;
    *) echo "direction must be 'forward' or 'reverse', got '{{direction}}'" >&2; exit 1 ;;
  esac
  api_s="LAB_${s}_API";  tok_s="LAB_${s}_TOKEN_ID"; sec_s="LAB_${s}_SECRET"; peer_s="LAB_${s}_PEER"
  api_d="LAB_${d}_API";  tok_d="LAB_${d}_TOKEN_ID"; sec_d="LAB_${d}_SECRET"; peer_d="LAB_${d}_PEER"
  export APIARY_CYCLE_SOURCE_API="${!api_s}"      APIARY_CYCLE_SOURCE_TOKEN_ID="${!tok_s}"
  export APIARY_CYCLE_SOURCE_SECRET="${!sec_s}"   APIARY_CYCLE_SOURCE_PEER="${!peer_s}"
  export APIARY_CYCLE_DEST_API="${!api_d}"        APIARY_CYCLE_DEST_TOKEN_ID="${!tok_d}"
  export APIARY_CYCLE_DEST_SECRET="${!sec_d}"     APIARY_CYCLE_DEST_PEER="${!peer_d}"
  export APIARY_CYCLE_MOVING="$LAB_MOVING"        APIARY_CYCLE_LINK="$LAB_LINK"
  echo "FLEET MOVE  ${LAB_MOVING#*,} :  cluster $from  ->  cluster $to"
  echo "=========================================================================="
  set -o pipefail
  go test ./internal/service/ -run TestFleetCycleAgainstLiveClusters \
    -count=1 -v -timeout 15m 2>&1 | python3 scripts/lab/timeline.py

# The departure gate, against real observations from cluster A.
#
# Non-destructive: it reads the provider and writes only to its own test schema.
# It never departs anything, so unlike `fleet-move` it is safe to run against a
# lab that is mid-experiment — which is the only way a refusal test gets run
# often enough to be worth having.
#
# Covers the refusals `fleet-move` structurally cannot: that test needs an empty
# node to get anywhere and stops on its own precondition when one is not, so
# guests_present — the refusal that stops a live host being torn out of its
# cluster — was never reached against real data.
[doc("Exercise the departure gate against live observations (non-destructive)")]
lab-gates: _lab-env _require-test-db
  #!/usr/bin/env bash
  set -euo pipefail
  set -a; . ./.env.lab; set +a
  set -a; . ./.env; set +a
  export TEST_DATABASE_URL="$(just _test-dsn)"
  unset GOROOT
  export APIARY_CYCLE_SOURCE_API="$LAB_A_API"
  export APIARY_CYCLE_SOURCE_TOKEN_ID="$LAB_A_TOKEN_ID"
  export APIARY_CYCLE_SOURCE_SECRET="$LAB_A_SECRET"
  echo "DEPARTURE GATE  vs cluster A"
  echo "=========================================================================="
  set -o pipefail
  go test ./internal/service/ -run TestDepartureGateAgainstLiveClusters \
    -count=1 -v -timeout 5m 2>&1 | python3 scripts/lab/timeline.py

# Scan every module for known vulnerabilities that this code can actually reach.
#
# govulncheck walks the CALL GRAPH, so it reports an advisory only when a
# vulnerable symbol is reachable from this code. That is the difference between
# it and a manifest scanner: an advisory in a dependency nothing calls is not a
# finding, so a non-empty result here is worth acting on rather than triaging.
#
# All four modules, because it analyses one at a time and the root run does not
# descend into the nested ones. GOWORK=off so each resolves through its own
# go.mod, which is how they are built and released.
#
# The database is fetched from vuln.go.dev at run time, so this can go red on a
# tree that has not changed — which is the point. It means an advisory was
# published against something already shipped.
[doc("Scan every Go module for reachable known vulnerabilities")]
vuln:
  #!/usr/bin/env bash
  set -euo pipefail
  # The toolchain is pinned in mise.toml; this is the fallback for a machine
  # that has not run `mise install`, and it is why the version appears twice.
  command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@v1.7.0
  failed=0
  for m in . ./pkg/proxmox ./sdk/go ./agent; do
    echo "--- $m"
    (cd "$m" && GOWORK=off govulncheck ./...) || failed=1
  done
  exit "$failed"
