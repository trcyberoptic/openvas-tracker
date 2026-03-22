---
name: deploy
description: Build frontend + Go binary and deploy to production server
---

Build and deploy OpenVAS-Tracker to production. Run these steps in order:

1. Build frontend:
   ```bash
   cd frontend && npm run build && cd ..
   ```

2. Copy frontend dist and cross-compile:
   ```bash
   rm -rf cmd/openvas-tracker/static && cp -r frontend/dist cmd/openvas-tracker/static
   GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/openvas-tracker-linux-amd64 ./cmd/openvas-tracker
   ```

3. Deploy to server:
   ```bash
   scp bin/openvas-tracker-linux-amd64 $DEPLOY_HOST:/usr/local/bin/openvas-tracker.new
   ssh $DEPLOY_HOST "chmod 755 /usr/local/bin/openvas-tracker.new && systemctl stop openvas-tracker && mv /usr/local/bin/openvas-tracker.new /usr/local/bin/openvas-tracker && systemctl start openvas-tracker"
   ```

4. Verify:
   ```bash
   sleep 2 && ssh $DEPLOY_HOST "curl -s http://localhost:8080/api/health"
   ```

Report the health check result to the user.
