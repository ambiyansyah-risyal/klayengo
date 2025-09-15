.PHONY: test bench clean lint cover fmt fmt-check

test:
	go test -race ./...

bench:
	go test -bench=. -benchmem ./...

clean:
	rm -f coverage.out

lint:
	golangci-lint run

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

fmt:
	go fmt ./...

fmt-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi
