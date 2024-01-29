.PHONY: start test benchmark

start:
	#rm $(PWD)/examples/logs/logfile.log
	cd examples && go run dev.go

test:
	@go test -coverprofile=coverage.out -coverpkg=`go list ./... | grep -v "/mock\|/examples"` .

cover:
	@go tool cover -html=coverage.out

benchmark:
	@go test ./... -bench . 

run-mock-server:
	@echo "Starting mock server..."
	@go run ./mock/server.go
