FROM postgres:19@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Extra databases created on first initialisation — see the script for why the
# Justfile also creates the Atlas one on demand.
COPY initdb/ /docker-entrypoint-initdb.d/
RUN chmod +x /docker-entrypoint-initdb.d/*.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["postgres", "-c", "ssl=on", "-c", "ssl_cert_file=/var/lib/postgresql/server.crt", "-c", "ssl_key_file=/var/lib/postgresql/server.key"]
