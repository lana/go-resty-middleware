.DEFAULT_GOAL := test

# Run tests and generates html coverage file
cover: test
	@go tool cover -html=./cover.out -o ./cover.html
	@rm ./cover.out
.PHONY: cover

# GolangCI Linter
lint:
	@golangci-lint run -v ./...
.PHONY: lint

# Run tests
test:
	@go test -v -covermode=atomic -coverprofile=cover.out ./...
.PHONY: test
