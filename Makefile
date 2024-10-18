.PHONY: all
all: | lint build test

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test -p 1 ./...
clean-db-raw:
	docker compose -f postgres/raw/compose.yml down --volumes --remove-orphans --rmi local

.PHONY: lint
lint:
	golangci-lint run

.PHONY: gen
gen:
	go generate ./...

.PHONY: check-gen
check-gen: gen
	git diff --quiet

.PHONY: run
run:
	go run github.com/calindra/nonodo
