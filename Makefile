.PHONY: build
build:
	@echo "Not implemented"
	exit 1


.PHONY: fmt
fmt:
	find . -name '*.go' | xargs -I{} go tool goimports -local 'github.com/high-moctane/mocjson-go' -w {}
	go tool golines -w .
	go tool gofumpt -w .


.PHONY: lint
lint:
	go vet ./...
	test -z "$$(go tool goimports -local 'github.com/high-moctane/mocjson-go' -l .)"
	test -z "$$(go tool golines -l .)"
	test -z "$$(go tool gofumpt -l .)"


.PHONY: test
test:
	go test -shuffle=on -cover -coverprofile=coverage.out ./...


.PHONY: bench
bench:
	go test -bench=. ./... -benchmem


.PHONY: analyze
analyze:
	go build -gcflags="-m -m" ./... 2> mocjson.analysis


.PHONY: asm
asm:
	go build -gcflags="-S" ./... 2> mocjson.asm


.PHONY: clean
clean:
	$(RM) coverage.out mocjson.analysis mocjson.asm
