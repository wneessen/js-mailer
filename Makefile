MODNAME		:= github.com/wneessen/js-mailer
SPACE		:= $(null) $(null)
CURVER		:= 0.1.0
BUILDVER	:= $(CURVER)
BUILDDIR	:= ./builds/$(CURVER)
LOCALOS     := $(shell uname -s)
LOCALARCH   := $(shell uname -m)

ifeq ($(OS), Windows_NT)
	OUTFILE	:= $(BUILDDIR)/js-mailer.exe
else
	OUTFILE	:= $(BUILDDIR)/js-mailer
endif

TARGETS			:= build-local
DOCKERTARGETS	:= build-prod-docker dockerize

all: $(TARGETS)

prod-docker: $(DOCKERTARGETS)


build-local:
	@echo "Building PROD version $(LOCALOS)/$(LOCALARCH) (OS on building machine)"
	@/usr/bin/env CGO_ENABLED=0 go build -o $(OUTFILE) -ldflags="-s -w" $(MODNAME)

build-prod-docker:
	/usr/bin/env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(OUTFILE) -ldflags="-s -w" $(MODNAME)

dockerize:
	@cat Dockerfile | sed "s/{{BUILDVER}}/$(CURVER)/g" >/var/tmp/Dockerfile_$(CURVER)
	@sudo docker build -t js-mailer:v$(CURVER) -f /var/tmp/Dockerfile_$(CURVER) .
	@echo "js-mailer v$(CURVER) image succesfully built as: js-mailer:v$(CURVER)"
	@mkdir -p docker-images/v$(CURVER)
	@sudo docker save -o docker-images/v$(CURVER)/js-mailer_v$(CURVER).img js-mailer:v$(CURVER)
	@sudo chown wneessen docker-images/v$(CURVER)/js-mailer_v$(CURVER).img
	@echo "js-mailer v$(CURVER) image succesfully exported to: docker-images/v$(CURVER)/js-mailer_v$(CURVER).img"