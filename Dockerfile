FROM scratch
LABEL maintainer="wn@neessen.net"
ENV RELEASE_VERSION={{BUILDVER}}
COPY ["build-files/passwd", "/etc/passwd"]
COPY ["build-files/group", "/etc/group"]
COPY --chown=js-mailer ["etc/js-mailer", "/etc/js-mailer/"]
COPY --chown=js-mailer ["builds/$RELEASE_VERSION/js-mailer", "/js-mailer/js-mailer"]
WORKDIR /js-mailer
USER js-mailer
VOLUME ["/etc/js-mailer"]
EXPOSE 8080
ENTRYPOINT ["/js-mailer/js-mailer"]