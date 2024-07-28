FROM debian:11-slim
RUN apt-get update && \
    apt-get install -y --no-install-suggests --no-install-recommends ca-certificates && \
    apt-get install -y curl && \
    apt-get clean && \
    groupadd -g 1001 serviceuser && \
    useradd -u 1001 -r -g 1001 -s /sbin/nologin -c "go service user" serviceuser

ADD /bin/anon3anon /app/bin/anon3anon
WORKDIR /app

CMD [ "/app/bin/anon3anon" ]
