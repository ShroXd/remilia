.PHONY: start test benchmark

start:
	#rm $(PWD)/examples/logs/logfile.log
	cd examples && go run dev.go

test:
	@go test -coverprofile=./out/coverage.out -coverpkg=`go list ./... | grep -v "/mock\|/examples"` .

cover:
	@go tool cover -html=./out/coverage.out

benchmark:
	@go test ./... -bench . 

run-mock-server:
	@echo "Starting mock server..."
	@go run ./mock/server.go
