FROM golang:1.18-alpine as golang

WORKDIR /go/src/app

COPY . .

# Static build required so that we can safely copy the binary over.
RUN CGO_ENABLED=0 go build -ldflags '-w -s'
RUN CGO_ENABLED=0 go build -ldflags '-w -s' ./cmd/gen_pass/gen_pass.go 

# ---
FROM alpine:latest as alpine

RUN apk --no-cache add tzdata zip

WORKDIR /usr/share/zoneinfo
# -0 means no compression.  Needed because go's
# tz loader doesn't handle compressed data.
RUN zip -q -r -0 /zoneinfo.zip .

# ---
FROM scratch

COPY --from=golang /go/src/app/export9p /
COPY --from=golang /go/src/app/gen_pass /

# the timezone data:
ENV ZONEINFO /zoneinfo.zip
ENV PATH /
COPY --from=alpine /zoneinfo.zip /

EXPOSE 14672
VOLUME /export

CMD ["export9p", "-dir", "/export"]
