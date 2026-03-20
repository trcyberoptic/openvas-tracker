# Dockerfile
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./cmd/openvas-tracker/static
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /openvas-tracker ./cmd/openvas-tracker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend /openvas-tracker /usr/local/bin/openvas-tracker
COPY sql/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["openvas-tracker"]
