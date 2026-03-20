.PHONY: build run test migrate-up migrate-down sqlc frontend dev clean

BINARY=vulntrack
BUILD_DIR=bin

build: frontend
	rm -rf cmd/vulntrack/static && cp -r frontend/dist cmd/vulntrack/static
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY) ./cmd/vulntrack

build-linux: frontend
	rm -rf cmd/vulntrack/static && cp -r frontend/dist cmd/vulntrack/static
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/vulntrack

run:
	go run ./cmd/vulntrack

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

migrate-up:
	migrate -path sql/migrations -database "$${DATABASE_URL}" up

migrate-down:
	migrate -path sql/migrations -database "$${DATABASE_URL}" down 1

sqlc:
	sqlc generate

frontend:
	cd frontend && npm ci && npm run build

dev:
	cd frontend && npm run dev &
	go run ./cmd/vulntrack

clean:
	rm -rf $(BUILD_DIR) frontend/dist cmd/vulntrack/static coverage.out
