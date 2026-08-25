#!/bin/bash
# Create the extra databases this project needs alongside the application one.
#
# Runs only on a FRESH data directory — the stock postgres entrypoint skips
# /docker-entrypoint-initdb.d entirely when the volume already contains a
# cluster. Anyone whose volume predates this file will not get these databases
# from here, which is why `just atlas-dev` also creates the Atlas one on demand.
# Both paths are idempotent and either alone is sufficient.
set -euo pipefail

# Space or comma separated. Defaults to the Atlas dev database.
EXTRA_DATABASES="${POSTGRES_EXTRA_DATABASES:-atlas_dev}"

create_database() {
	local db="$1"
	# Not "CREATE DATABASE IF NOT EXISTS" — Postgres has no such form. Ask the
	# catalog first and skip if present, so re-running is harmless.
	if psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
		-tAc "SELECT 1 FROM pg_database WHERE datname = '${db}'" | grep -q 1; then
		echo "database '${db}' already exists, leaving it alone"
		return
	fi
	echo "creating database '${db}'"
	psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-SQL
		CREATE DATABASE "${db}" OWNER "${POSTGRES_USER}";
	SQL
}

for db in ${EXTRA_DATABASES//,/ }; do
	[ -z "$db" ] && continue
	# Refuse to touch the application database. Atlas treats its dev database as
	# scratch — it creates, rewrites and drops objects in it on every run — so
	# naming the app database here would hand a migration tool a live schema to
	# wipe.
	if [ "$db" = "$POSTGRES_DB" ]; then
		echo "refusing to list the application database '${db}' as a scratch database" >&2
		exit 1
	fi
	create_database "$db"
done
