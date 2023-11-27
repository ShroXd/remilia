.PHONY: start test benchmark

start:
	#rm $(PWD)/examples/logs/logfile.log
	cd examples && go run dev.go -race

test:
	@go test -cover ./...

benchmark:
	@go test ./... -bench . 

run-mock-server:
	@echo "Starting mock server..."
	@go run ./mock/server.go
