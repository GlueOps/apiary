FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

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
