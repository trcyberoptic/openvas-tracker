#!/usr/bin/env bash
# deploy/install.sh — Install VulnTrack Pro on Debian Trixie
set -euo pipefail

BINARY_SRC="${1:-bin/vulntrack-linux-amd64}"

echo "==> Creating vulntrack user"
useradd --system --shell /usr/sbin/nologin --home-dir /var/lib/vulntrack vulntrack 2>/dev/null || true

echo "==> Installing binary"
install -o root -g root -m 0755 "$BINARY_SRC" /usr/local/bin/vulntrack

echo "==> Creating config directory"
install -d -o vulntrack -g vulntrack -m 0750 /etc/vulntrack
install -d -o vulntrack -g vulntrack -m 0750 /var/lib/vulntrack

if [ ! -f /etc/vulntrack/env ]; then
    install -o vulntrack -g vulntrack -m 0600 deploy/vulntrack.env.example /etc/vulntrack/env
    echo "==> Created /etc/vulntrack/env — EDIT THIS FILE with your secrets"
fi

echo "==> Installing systemd unit"
install -o root -g root -m 0644 deploy/vulntrack.service /etc/systemd/system/vulntrack.service
systemctl daemon-reload

echo "==> Running database migrations"
su -s /bin/bash vulntrack -c "VT_DATABASE_URL=\$(grep VT_DATABASE_URL /etc/vulntrack/env | cut -d= -f2-) /usr/local/bin/vulntrack migrate"

echo "==> Enabling and starting service"
systemctl enable vulntrack
systemctl start vulntrack
systemctl status vulntrack

echo "==> Done! VulnTrack Pro is running on port 8080"
