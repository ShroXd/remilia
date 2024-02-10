BINARY_NAME=lib
OUTPUT_DIR=out
BENCHMARK_DIR=benchmarks
COVER_PROFILE=$(OUTPUT_DIR)/coverage.out
CPU_PROFILE=$(OUTPUT_DIR)/cpu.pprof
MEM_PROFILE=$(OUTPUT_DIR)/mem.pprof
BLOCK_PROFILE=$(OUTPUT_DIR)/block.pprof
GOROUTINE_PROFILE=$(OUTPUT_DIR)/goroutine.pprof
MUTEX_PROFILE=$(OUTPUT_DIR)/mutex.pprof
THREADCREATE_PROFILE=$(OUTPUT_DIR)/threadcreate.pprof
TRACE_PROFILE=$(OUTPUT_DIR)/trace.pprof

$(OUTPUT_DIR) $(BENCHMARK_DIR):
	@echo "📂 Creating directory $@"
	@mkdir -p $@

build: $(OUTPUT_DIR)
	@echo "🚀 Building the project..."
	go build -o $(OUTPUT_DIR)/$(BINARY_NAME) ./examples/dev.go

run: build
	@echo "🏃‍♀️ Running the binary..."
	@./$(OUTPUT_DIR)/$(BINARY_NAME)

test: $(OUTPUT_DIR)
	@echo "🧪 Running tests..."
	go test -coverprofile=$(COVER_PROFILE) -coverpkg=`go list ./... | grep -v "/mock\|/examples"`

cover: test
	@echo "📊 Generating coverage report..."
	@go tool cover -html=$(COVER_PROFILE)

benchmark: $(OUTPUT_DIR)
	@echo "⚖️ running benchmarks..."
	@branch=$$(git rev-parse --abbrev-ref head); \
	commit=$$(git rev-parse --short head); \
	tag=$$(git describe --tags --exact-match $$commit 2>/dev/null); \
	file_tag=$${tag:+_$${tag}}; \
	file_name="benchmarks/bench_$${branch}_$${commit}$${file_tag}.txt"; \
	go list ./... | grep -v "/examples" | grep -v "/mock" | xargs go test -bench . -benchmem -cpuprofile $(CPU_PROFILE) -memprofile $(MEM_PROFILE) | tee $$file_name

profile: build
	@echo "📈 Running the program and collecting performance data..."
	@./$(OUTPUT_DIR)/$(BINARY_NAME) 2> /dev/null
	@echo "📊 CPU and memory profile files generated in $(OUTPUT_DIR)/"

view-cpu:
	@echo "👁️‍🗨️ Viewing CPU profile file..."
	@go tool pprof -http=:8080 $(CPU_PROFILE)

view-mem:
	@echo "👁️‍🗨️ Viewing memory profile file..."
	@go tool pprof -http=:8080 $(MEM_PROFILE)

view-block:
	@echo "👁️‍🗨️ Viewing block profile file..."
	@go tool pprof -http=:8080 $(BLOCK_PROFILE)

view-goroutine:
	@echo "👁️‍🗨️ Viewing goroutine profile file..."
	@go tool pprof -http=:8080 $(GOROUTINE_PROFILE)

view-mutex:
	@echo "👁️‍🗨️ Viewing mutex profile file..."
	@go tool pprof -http=:8080 $(MUTEX_PROFILE)

view-threadcreate:
	@echo "👁️‍🗨️ Viewing threadcreate profile file..."
	@go tool pprof -http=:8080 $(THREADCREATE_PROFILE)

view-trace:
	@echo "👁️‍🗨️ Viewing trace file..."
	@go tool trace $(TRACE_PROFILE)

clean:
	@echo "🧹 Cleaning up..."
	@rm -rf $(OUTPUT_DIR)
	@echo "✅ Cleaned up"

.PHONY: build run test cover benchmark profile view-cpu view-mem clean
