.PHONY: test lint fmt tidy vet

test:
	go test -race -shuffle=on ./...

vet:
	go vet ./...

lint: vet
	golangci-lint run

fmt:
	gofmt -s -w .

tidy:
	go mod tidy
