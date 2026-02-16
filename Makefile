.PHONY: build run test clean keygen client

build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/client cmd/client/main.go
	go build -o bin/keygen cmd/keygen/main.go

run:
	go run cmd/server/main.go

client:
	go run cmd/client/main.go -password $(PASSWORD) -mode $(MODE)

keygen:
	go run cmd/keygen/main.go -password $(PASSWORD) -out $(OUT)

test:
	go test -v ./...

clean:
	rm -rf bin/
	rm -f *.db
	rm -f *.enc