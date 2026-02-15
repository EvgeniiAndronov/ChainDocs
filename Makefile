.PHONY: build run test clean

build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/client cmd/client/main.go

run:
	go run cmd/server/main.go

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean:
	rm -rf bin/
	rm -f *.db
	rm -f coverage.out

deps:
	go mod tidy
	go mod download

# Запуск с горячей перезагрузкой (требуется air)
dev:
	air -c .air.toml

# Создать тестовый блок
test-block:
	curl -X POST http://localhost:8080/api/blocks \
		-H "Content-Type: application/json" \
		-d '{"document_hash":"0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"}'

.PHONY: build run test test-coverage clean deps dev test-block