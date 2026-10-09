# Build image for Anthropic's OSS Scanner (https://github.com/anthropics/oss-scanner).
# The scanner builds this with the repository as context, then runs its analysis with NO network,
# so everything the build, the tests and a local test instance need is fetched here.
# Trixie ships MariaDB 11.8, the version production runs.
FROM golang:1.26-trixie

# golang images set GOTOOLCHAIN=local; auto lets go.mod pull its exact patch release (cached for offline use).
ENV DEBIAN_FRONTEND=noninteractive \
    GOTOOLCHAIN=auto \
    CGO_ENABLED=0

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git jq make python3 python3-bcrypt \
      nodejs npm mariadb-server mariadb-client \
    && rm -rf /var/lib/apt/lists/*

# scanner contract: the checkout lives inside the image, at /src
COPY . /src
WORKDIR /src

# Go modules + toolchain, frontend packages (incl. eslint/tsc devDependencies).
RUN go mod download && go version \
    && cd frontend && npm ci --no-audit --no-fund

# Production build exactly as `make build`: frontend → embedded static/, binary → bin/openvas-tracker.
# -gcflags keeps the binary debuggable (no -s -w, no inlining).
RUN npm --prefix frontend run build \
    && rm -rf /src/cmd/openvas-tracker/static && cp -r /src/frontend/dist /src/cmd/openvas-tracker/static \
    && go build -gcflags="all=-N -l" -o /src/bin/openvas-tracker ./cmd/openvas-tracker \
    && go vet ./...

# Compile every test binary now so `go test` works offline; a failing test must not abort the build.
RUN go test ./... -count=1 || echo "WARNING: go test reported failures"

# Local test instance (MariaDB + app with throwaway secrets): see .oss-scanner/threat_model.md.
RUN install -m 0755 /src/.oss-scanner/start-tracker.sh /usr/local/bin/start-tracker
