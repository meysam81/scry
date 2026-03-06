build:
	go build

test:
	go test -race ./...

lint:
	golangci-lint run ./...

run-example:
	go run . crawl https://httpbin.org --output terminal
