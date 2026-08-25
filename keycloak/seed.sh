#!/usr/bin/env bash

set -euo pipefail

KCADM="/opt/keycloak/bin/kcadm.sh"
REALM_FILE="/seed/realm.json"

echo "Authenticating with Keycloak..."

"${KCADM}" config credentials \
  --server "${KEYCLOAK_URL}" \
  --realm master \
  --user "${KEYCLOAK_ADMIN}" \
  --password "${KEYCLOAK_ADMIN_PASSWORD}"

# -s realm=... on both branches: the name in realm.json and $KEYCLOAK_REALM are
# two places to say the same thing, and if they disagree every run creates the
# file's realm and then looks for the env one, forever.
if "${KCADM}" get "realms/${KEYCLOAK_REALM}" >/dev/null 2>&1; then
  echo "Realm '${KEYCLOAK_REALM}' exists; applying realm settings..."

  "${KCADM}" update "realms/${KEYCLOAK_REALM}" \
    -s "realm=${KEYCLOAK_REALM}" \
    -f "${REALM_FILE}"
else
  echo "Creating realm '${KEYCLOAK_REALM}'..."

  "${KCADM}" create realms \
    -s "realm=${KEYCLOAK_REALM}" \
    -f "${REALM_FILE}"
fi

# The realm update above quietly drops the clients and users arrays — only a
# realm *import* reads them. So they are applied separately, on every run, or
# the client added to realm.json today would never appear on a realm created
# yesterday. OVERWRITE is what makes it converge rather than skip: a changed
# secret or a new redirect URI replaces what is there.
echo "Applying clients and users..."

"${KCADM}" create partialImport \
  -r "${KEYCLOAK_REALM}" \
  -s ifResourceExists=OVERWRITE \
  -o \
  -f "${REALM_FILE}"

echo "Realm '${KEYCLOAK_REALM}' is up to date."
