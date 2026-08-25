#!/bin/sh
set -e

KEY_FILE=/etc/pgbouncer/server.key
CERT_FILE=/etc/pgbouncer/server.crt

KEY_EXISTS=0
CERT_EXISTS=0
[ -f "$KEY_FILE" ]  && KEY_EXISTS=1
[ -f "$CERT_FILE" ] && CERT_EXISTS=1

if [ "$KEY_EXISTS" -eq 1 ] && [ "$CERT_EXISTS" -eq 1 ]; then
    # Both files are present — verify they are readable and the key is not over-permissive.
    if [ ! -r "$KEY_FILE" ]; then
        echo "ERROR: $KEY_FILE exists but is not readable by the current user." >&2
        echo "       Check ownership and permissions (expected: pgbouncer:pgbouncer, mode 400 or 600)." >&2
        exit 1
    fi
    KEY_PERMS=$(stat -c '%a' "$KEY_FILE")
    if [ "$KEY_PERMS" != "400" ] && [ "$KEY_PERMS" != "600" ]; then
        echo "ERROR: $KEY_FILE has permissions $KEY_PERMS — must be 400 or 600." >&2
        echo "       Private key must not be readable by group or world (e.g. chmod 600 $KEY_FILE)." >&2
        exit 1
    fi
    if [ ! -r "$CERT_FILE" ]; then
        echo "ERROR: $CERT_FILE exists but is not readable by the current user." >&2
        echo "       Check ownership and permissions (expected: pgbouncer:pgbouncer, mode 644)." >&2
        exit 1
    fi
    CERT_PERMS=$(stat -c '%a' "$CERT_FILE")
    if [ "$CERT_PERMS" != "644" ]; then
        echo "ERROR: $CERT_FILE has permissions $CERT_PERMS — must be 644." >&2
        echo "       Certificate should be world-readable but not writable (e.g. chmod 644 $CERT_FILE)." >&2
        exit 1
    fi
elif [ "$KEY_EXISTS" -eq 0 ] && [ "$CERT_EXISTS" -eq 0 ]; then
    # Neither file is present — generate a self-signed certificate.
    # To supply your own certificates, mount them before container start:
    #   /etc/pgbouncer/server.key  (mode 400 or 600, owner pgbouncer)
    #   /etc/pgbouncer/server.crt  (mode 644, owner pgbouncer)
    echo "No TLS certificates found — generating a self-signed certificate for pgbouncer."
    # Use a restrictive umask so the key file is never transiently world-readable.
    (
        umask 077
        openssl req -x509 -nodes -newkey rsa:4096 \
            -keyout "$KEY_FILE" \
            -out "$CERT_FILE" \
            -days 365 \
            -subj "/CN=localhost" \
            -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
    )
    chmod 600 "$KEY_FILE"
    chmod 644 "$CERT_FILE"
else
    # Exactly one file is present — this is a misconfiguration; refuse to
    # overwrite or silently proceed, as this could corrupt a partial mount.
    if [ "$KEY_EXISTS" -eq 0 ]; then
        MISSING="$KEY_FILE"
        PRESENT="$CERT_FILE"
    elif [ "$CERT_EXISTS" -eq 0 ]; then
        MISSING="$CERT_FILE"
        PRESENT="$KEY_FILE"
    fi
    echo "ERROR: TLS certificate mismatch — $PRESENT is present but $MISSING is missing." >&2
    echo "       Mount both files or neither. Refusing to overwrite existing material." >&2
    exit 1
fi

exec "$@"
