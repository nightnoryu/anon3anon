FROM alpine:latest

ADD /bin/anon3anon /app/bin/anon3anon
WORKDIR /app

EXPOSE 8080
CMD [ "/app/bin/anon3anon" ]
