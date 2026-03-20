# Dockerfile
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /vulntrack ./cmd/vulntrack

FROM alpine:3.20
RUN apk add --no-cache ca-certificates nmap
COPY --from=backend /vulntrack /usr/local/bin/vulntrack
COPY sql/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["vulntrack"]
