.PHONY: build
build:
	@echo "Not implemented"
	exit 1


.PHONY: fmt
fmt:
	go fmt ./...


.PHONY: test
test:
	go test -shuffle=on ./...


.PHONY: bench
bench:
	go test -bench=. ./... -benchmem


.PHONY: analyze
analyze:
	go build -gcflags="-m" ./...
