start:
	cd examples && go run dev.go

.PHONY: test
test:
	@go test -cover ./...
