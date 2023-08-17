.PHONY: start test benchmark

start:
	cd examples && go run dev.go

test:
	@go test -cover ./...

benchmark:
	@go test ./... -bench . 
