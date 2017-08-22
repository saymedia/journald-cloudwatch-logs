GOOS?=linux
GOARCH?=amd64
SOURCES:=$(shell find -type f -name '*.go')

PROGRAM:=journald-cloudwatch-logs

.PHONY: all
all: $(PROGRAM)

.PHONY: deps
deps:
	go get

$(PROGRAM): $(SOURCES) deps
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@

.PHONY: install
install: build
	go install

.PHONY: clean
clean:
	-rm $(PROGRAM)
