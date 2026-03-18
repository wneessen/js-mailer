# SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
#
# SPDX-License-Identifier: MIT

## Build first
FROM --platform=${BUILDPLATFORM} golang:latest@sha256:c42e4d75186af6a44eb4159dcfac758ef1c05a7011a0052fe8a8df016d8e8fb9 AS builder
RUN mkdir /builddir
ADD . /builddir/
WORKDIR /builddir
RUN go mod tidy && go mod download && go mod verify

ARG VERSION
ARG COMMIT
ARG DATE
ARG BUILDARCH
ARG TARGETARCH
RUN GOOS=linux GOARCH=${TARGETARCH} CGO_ENABLED=0 go build \
    -a -installsuffix cgo \
    -ldflags "-w -s -extldflags \"-static\" -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o js-mailer github.com/wneessen/js-mailer/cmd/js-mailer

## Create scratch image
FROM scratch
LABEL maintainer="wn@neessen.dev"
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
