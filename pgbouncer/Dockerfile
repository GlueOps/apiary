FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache \
    pgbouncer \
    openssl \
    libevent \
    postgresql-client && \
    addgroup -S pgbouncer && \
    adduser -S -G pgbouncer pgbouncer && \
    mkdir -p /etc/pgbouncer /var/log/pgbouncer /var/run/pgbouncer

COPY --chown=pgbouncer:pgbouncer pgbouncer.ini /etc/pgbouncer/pgbouncer.ini
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN touch /etc/pgbouncer/userlist.txt && \
    chown -R pgbouncer:pgbouncer /etc/pgbouncer /var/log/pgbouncer /var/run/pgbouncer && \
    chmod 700 /etc/pgbouncer

EXPOSE 6432

USER pgbouncer

ENTRYPOINT ["/entrypoint.sh"]
CMD ["pgbouncer", "/etc/pgbouncer/pgbouncer.ini"]
