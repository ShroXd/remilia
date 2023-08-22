.PHONY: start test benchmark

start:
	#rm $(PWD)/examples/logs/logfile.log
	cd examples && go run dev.go

test:
	@go test -cover ./...

benchmark:
	@go test ./... -bench . 
