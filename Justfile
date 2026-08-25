generate name:
  go run . generate migrations {{name}}

sdk:
  go run . generate sdk

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
