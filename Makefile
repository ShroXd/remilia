BINARY_NAME=lib
OUTPUT_DIR=out
COVER_PROFILE=$(OUTPUT_DIR)/coverage.out
CPU_PROFILE=$(OUTPUT_DIR)/cpu.pprof
MEM_PROFILE=$(OUTPUT_DIR)/mem.pprof

$(OUTPUT_DIR):
	@echo "📂 Creating output directory..."
	@mkdir -p $(OUTPUT_DIR)

build: $(OUTPUT_DIR)
	@echo "🚀 Building the project..."
	@go build -o $(OUTPUT_DIR)/$(BINARY_NAME) ./examples/dev.go

run: build
	@echo "🏃‍♀️ Running the binary..."
	@$(OUTPUT_DIR)/$(BINARY_NAME)

test: $(OUTPUT_DIR)
	@echo "🧪 Running tests..."
	@go test -coverprofile=$(COVER_PROFILE) -coverpkg=`go list ./... | grep -v "/mock\|/examples"` .

cover:
	@echo "📊 Generating coverage report..."
	@go tool cover -html=$(COVER_PROFILE)

benchmark:
	@echo "⚖️ Running benchmarks..."
	@go test ./... -bench .

profile: build
	@echo "📈 Running the program and collecting performance data..."
	@./$(OUTPUT_DIR)/$(BINARY_NAME) 2> /dev/null
	@echo "📊 CPU and memory profile files generated in $(OUTPUT_DIR)/"

view-cpu: profile 
	@echo "👁️‍🗨️ Viewing CPU profile file..."
	@go tool pprof -http=:8080 $(CPU_PROFILE)

view-mem: profile 
	@echo "👁️‍🗨️ Viewing memory profile file..."
	@go tool pprof -http=:8080 $(MEM_PROFILE)

clean:
	@echo "🧹 Cleaning up..."
	@rm -rf $(OUTPUT_DIR)
	@echo "✅ Cleaned up"

.PHONY: build run test cover benchmark profile view-cpu view-mem clean
