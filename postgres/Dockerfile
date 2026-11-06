FROM postgres:18@sha256:e3d24f6906d0760bf4da0dae872415391ededefad23fd471c47dc53b0ddf3c42

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Extra databases created on first initialisation — see the script for why the
# Justfile also creates the Atlas one on demand.
COPY initdb/ /docker-entrypoint-initdb.d/
RUN chmod +x /docker-entrypoint-initdb.d/*.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["postgres", "-c", "ssl=on", "-c", "ssl_cert_file=/var/lib/postgresql/server.crt", "-c", "ssl_key_file=/var/lib/postgresql/server.key"]
