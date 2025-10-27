## Build first
FROM golang:latest@sha256:6bac879c5b77e0fc9c556a5ed8920e89dab1709bd510a854903509c828f67f96 as builder
RUN mkdir /builddir
ADD . /builddir/
WORKDIR /builddir
RUN go mod tidy && go mod download && go mod verify
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"' -o js-mailer \
    github.com/wneessen/js-mailer

## Create scratch image
FROM scratch
LABEL maintainer="wn@neessen.net"
COPY ["docker-files/passwd", "/etc/passwd"]
COPY ["docker-files/group", "/etc/group"]
COPY --from=builder ["/etc/ssl/certs/ca-certificates.crt", "/etc/ssl/cert.pem"]
COPY --chown=js-mailer ["LICENSE", "/js-mailer/LICENSE"]
COPY --chown=js-mailer ["README.md", "/js-mailer/README.md"]
COPY --chown=js-mailer ["etc/js-mailer", "/etc/js-mailer/"]
COPY --from=builder --chown=js-mailer ["/builddir/js-mailer", "/js-mailer/js-mailer"]
WORKDIR /js-mailer
USER js-mailer
VOLUME ["/etc/js-mailer"]
EXPOSE 8765
ENTRYPOINT ["/js-mailer/js-mailer"]
