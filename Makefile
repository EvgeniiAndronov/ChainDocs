.PHONY: build run test clean keygen client demo-start demo-stop demo-clean install help

# ==================== BUILD ====================
build: build-server build-client build-keygen build-signer
	@echo "✅ All binaries built"

build-server:
	@echo "🔨 Building server..."
	go build -o bin/server ./cmd/server/main.go ./cmd/server/config.go

build-client:
	@echo "🔨 Building client..."
	go build -o bin/client ./cmd/client/main.go ./cmd/client/config.go

build-keygen:
	@echo "🔨 Building keygen..."
	go build -o bin/keygen ./cmd/keygen/main.go

build-signer:
	@echo "🔨 Building signer..."
	go build -o bin/signer ./cmd/signer/main.go

# ==================== RUN ====================
run:
	go run cmd/server/main.go

client:
	@echo "Usage: make client PASSWORD=your_password MODE=oneshot|daemon"
	go run cmd/client/main.go -password $(PASSWORD) -mode $(MODE)

# ==================== TEST ====================
test: test-unit test-integration
	@echo "✅ All tests completed"

test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -race ./internal/... ./cmd/...

test-integration:
	@echo "🧪 Running integration tests..."
	go test -v -timeout 5m ./test/integration/...

test-live: build
	@echo "🧪 Running live tests..."
	chmod +x test-live.sh
	./test-live.sh

test-all: test-unit test-integration test-live
	@echo "✅ All tests passed!"

# ==================== DEMO ====================
demo-start: build
	@echo "🚀 Starting demo environment..."
	chmod +x demo/demo-start.sh
	./demo/demo-start.sh

demo-stop:
	@echo "🛑 Stopping demo environment..."
	chmod +x demo/demo-stop.sh
	./demo/demo-stop.sh

demo-clean:
	@echo "🧹 Cleaning demo environment..."
	chmod +x demo/demo-cleanup.sh
	./demo/demo-cleanup.sh

demo-restart: demo-stop demo-start
	@echo "🔄 Demo restarted"

# ==================== KEYGEN ====================
keygen:
	@echo "🔑 Generating key pair..."
	go run cmd/keygen/main.go -password $(PASSWORD) -out $(OUT)

# ==================== INSTALL ====================
install: build
	@echo "📦 Installing ChainDocs..."
	chmod +x scripts/install/install-client.sh
	sudo ./scripts/install/install-client.sh -b ./bin/client -d

uninstall:
	@echo "🗑️  Uninstalling ChainDocs..."
	sudo ./scripts/install/install-client.sh --uninstall

# ==================== CLEAN ====================
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -f *.db
	rm -f *.enc
	rm -f coverage.out
	@echo "✅ Clean complete"

clean-all: clean demo-clean
	@echo "✅ Full clean complete"

# ==================== DOCKER ====================
docker-build:
	@echo "🐳 Building Docker images..."
	docker build -t chaindocs-server:latest .
	docker build -f Dockerfile.client -t chaindocs-client:latest .

docker-up:
	@echo "🐳 Starting Docker containers..."
	docker-compose up -d

docker-down:
	@echo "🐳 Stopping Docker containers..."
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-clean:
	@echo "🐳 Cleaning Docker artifacts..."
	docker-compose down -v
	docker system prune -f

# ==================== LINT ====================
lint:
	@echo "🔍 Running linters..."
	gofmt -l .
	go vet ./...

lint-fix:
	@echo "🔧 Fixing code formatting..."
	go fmt ./...

# ==================== HELP ====================
help:
	@echo "ChainDocs Makefile Commands"
	@echo ""
	@echo "🔨 Build:"
	@echo "  make build          - Build all binaries"
	@echo "  make build-server   - Build server only"
	@echo "  make build-client   - Build client only"
	@echo "  make build-keygen   - Build keygen only"
	@echo ""
	@echo "🏃 Run:"
	@echo "  make run            - Run server locally"
	@echo "  make client PASSWORD=x MODE=y - Run client (oneshot|daemon)"
	@echo ""
	@echo "🧪 Test:"
	@echo "  make test           - Run all tests"
	@echo "  make test-unit      - Run unit tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-live      - Run live tests"
	@echo ""
	@echo "🎯 Demo:"
	@echo "  make demo-start     - Start demo environment"
	@echo "  make demo-stop      - Stop demo environment"
	@echo "  make demo-clean     - Clean demo data"
	@echo "  make demo-restart   - Restart demo"
	@echo ""
	@echo "🔑 Keygen:"
	@echo "  make keygen PASSWORD=x OUT=y - Generate key pair"
	@echo ""
	@echo "📦 Install:"
	@echo "  make install        - Install client as daemon"
	@echo "  make uninstall      - Uninstall client"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make docker-build   - Build Docker images"
	@echo "  make docker-up      - Start containers"
	@echo "  make docker-down    - Stop containers"
	@echo "  make docker-logs    - View logs"
	@echo ""
	@echo "🧹 Clean:"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make clean-all      - Clean everything including demo"
	@echo ""
	@echo "🔍 Other:"
	@echo "  make lint           - Run linters"
	@echo "  make lint-fix       - Fix code formatting"
	@echo "  make help           - Show this help"
