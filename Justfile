build:
  go build

test:
  go test -race ./...

run-example:
  go run . crawl https://httpbin.org --output terminal

check:
  golangci-lint run ./...
  go test ./...
