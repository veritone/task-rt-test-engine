FROM alpine:3.5

ARG APP_ENV=local
ENV APP_ENV ${APP_ENV}

ENV CONFIG_PATH /app/config.json

ADD . /go/src/github.com/veritone/task-rt-test-engine

ADD task-rt-test-engine /app/
ADD config/${APP_ENV}.json /app/config.json

ADD manifest.json /var/manifest.json

ENTRYPOINT ["/app/task-rt-test-engine"]
