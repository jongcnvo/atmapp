
GOBIN = $(shell pwd)/build/bin
GO ?= latest

atmapp:
	build/env.sh go run build/ci.go install ./app
	@echo "Done building."
	@echo "Run \"$(GOBIN)/atmapp\" to launch ATMChain."