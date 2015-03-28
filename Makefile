GO=go
GOPATH=$(shell pwd)
GOFLAGS=install
BIN=logger

all : $(BIN)

logger : src/go_logger/logger.go
	GOPATH=$(GOPATH) $(GO) $(GOFLAGS) go_logger

clean :
	-@ rm -rf $(BIN)
