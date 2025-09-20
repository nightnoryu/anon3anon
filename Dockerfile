FROM alpine:latest

ADD /bin/anon3anon /app/bin/anon3anon
WORKDIR /app

CMD [ "/app/bin/anon3anon" ]
