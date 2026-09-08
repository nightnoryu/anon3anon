FROM alpine:3.23

RUN addgroup -g 10001 -S anon3anon \
    && adduser -u 10001 -S -G anon3anon -H -s /sbin/nologin anon3anon \
    && mkdir -p /data \
    && chown anon3anon:anon3anon /data

COPY --chown=anon3anon:anon3anon /bin/anon3anon /app/bin/anon3anon

WORKDIR /app
USER 10001:10001

EXPOSE 8080
CMD [ "/app/bin/anon3anon" ]
