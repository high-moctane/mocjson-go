.PHONY: build
build:
	@echo "Not implemented"
	exit 1


.PHONY: fmt
fmt:
	goimports -local 'github.com/high-moctane/mocjson-go' -w .


.PHONY: lint
lint:
	go vet ./...
	test -z "$$(goimports -local 'github.com/high-moctane/mocjson-go' -l .)"


.PHONY: test
test:
	go test -shuffle=on -cover -coverprofile=coverage.out ./...


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
	$(RM) coverage.out mocjson.analysis mocjson.asm
