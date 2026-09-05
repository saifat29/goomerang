FROM alpine:3.24.1

RUN apk add --no-cache --no-progress ca-certificates tzdata

ARG TARGETPLATFORM
COPY ./dist/$TARGETPLATFORM/goomerang /

EXPOSE 8080

ENTRYPOINT ["/goomerang"]
