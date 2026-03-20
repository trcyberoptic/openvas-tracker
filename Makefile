.PHONY: build run test migrate-up migrate-down sqlc frontend dev clean

BINARY=openvas-tracker
BUILD_DIR=bin

build: frontend
	rm -rf cmd/openvas-tracker/static && cp -r frontend/dist cmd/openvas-tracker/static
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY) ./cmd/openvas-tracker

build-linux: frontend
	rm -rf cmd/openvas-tracker/static && cp -r frontend/dist cmd/openvas-tracker/static
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/openvas-tracker

run:
	go run ./cmd/openvas-tracker

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
	go run ./cmd/openvas-tracker

clean:
	rm -rf $(BUILD_DIR) frontend/dist cmd/openvas-tracker/static coverage.out
