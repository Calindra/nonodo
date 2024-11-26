.PHONY: all
all: | lint build test

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test --timeout 1m -p 1 ./...
clean-db-raw:
	docker compose -f postgres/raw/compose.yml down --volumes --remove-orphans --rmi local
up-db-raw:
	docker compose -f postgres/raw/compose.yml up --wait
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
