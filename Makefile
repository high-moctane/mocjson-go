.PHONY: build
build:
	@echo "Not implemented"
	exit 1


.PHONY: fmt
fmt:
	goimports -local 'github.com/high-moctane/mocjson-go' -w .


.PHONY: test
test:
	go test -shuffle=on ./...


.PHONY: bench
bench:
	go test -bench=. ./... -benchmem


.PHONY: analyze
analyze:
	go build -gcflags="-m" ./... 2> mocjson.analysis


.PHONY: asm
asm:
	go build -gcflags="-S" ./... 2> mocjson.asm


.PHONY: clean
clean:
	$(RM) mocjson.analysis mocjson.asm
