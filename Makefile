.PHONY: test bench clean lint

test:
	go test -race ./...

bench:
	go test -bench=. -benchmem ./...

clean:
	rm -f coverage.out

lint:
	golangci-lint run
