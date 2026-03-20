#!/usr/bin/env bash
# deploy/install.sh — Install OpenVAS-Tracker on Debian Trixie
set -euo pipefail

BINARY_SRC="${1:-bin/openvas-tracker-linux-amd64}"

echo "==> Creating openvas-tracker user"
useradd --system --shell /usr/sbin/nologin --home-dir /var/lib/openvas-tracker openvas-tracker 2>/dev/null || true

echo "==> Installing binary"
install -o root -g root -m 0755 "$BINARY_SRC" /usr/local/bin/openvas-tracker

echo "==> Creating config directory"
install -d -o openvas-tracker -g openvas-tracker -m 0750 /etc/openvas-tracker
install -d -o openvas-tracker -g openvas-tracker -m 0750 /var/lib/openvas-tracker

if [ ! -f /etc/openvas-tracker/env ]; then
    install -o openvas-tracker -g openvas-tracker -m 0600 deploy/openvas-tracker.env.example /etc/openvas-tracker/env
    echo "==> Created /etc/openvas-tracker/env — EDIT THIS FILE with your secrets"
fi

echo "==> Installing systemd unit"
install -o root -g root -m 0644 deploy/openvas-tracker.service /etc/systemd/system/openvas-tracker.service
systemctl daemon-reload

echo "==> Running database migrations"
su -s /bin/bash openvas-tracker -c "OT_DATABASE_URL=\$(grep OT_DATABASE_URL /etc/openvas-tracker/env | cut -d= -f2-) /usr/local/bin/openvas-tracker migrate"

echo "==> Enabling and starting service"
systemctl enable openvas-tracker
systemctl start openvas-tracker
systemctl status openvas-tracker

echo "==> Done! OpenVAS-Tracker is running on port 8080"
