## Build first
FROM golang:alpine as builder
RUN mkdir /builddir
ADD . /builddir/
WORKDIR /builddir
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"' -o js-mailer \
    github.com/wneessen/js-mailer

## Create scratch image
FROM scratch
LABEL maintainer="wn@neessen.net"
COPY ["build-files/passwd", "/etc/passwd"]
COPY ["build-files/group", "/etc/group"]
COPY --from=builder ["/etc/ssl/certs/ca-certificates.crt", "/etc/ssl/cert.pem"]
COPY --chown=js-mailer ["etc/js-mailer", "/etc/js-mailer/"]
COPY --from=builder --chown=js-mailer ["/builddir/js-mailer", "/js-mailer/js-mailer"]
WORKDIR /js-mailer
USER js-mailer
VOLUME ["/etc/js-mailer"]
EXPOSE 8765
ENTRYPOINT ["/js-mailer/js-mailer"]