#!/bin/sh
# Starts MariaDB and openvas-tracker inside the OSS Scanner image, with throwaway credentials
# that exist only in this image. Idempotent: a second run only reprints the credentials.
#
#   start-tracker            # app on http://127.0.0.1:8080, logs in /tmp/openvas-tracker.log
set -eu

ADMIN_PASSWORD=scanner-admin-password-0123456789
VIEWER_PASSWORD=scanner-viewer-password-012345678
IMPORT_APIKEY=scanner-import-apikey-0123456789abcdef
BIN=${BIN:-/src/bin/openvas-tracker}

if ! mariadb-admin ping --silent >/dev/null 2>&1; then
    mkdir -p /run/mysqld && chown mysql:mysql /run/mysqld
    mariadbd-safe --user=mysql >/tmp/mariadb.log 2>&1 &
    i=0
    until mariadb-admin ping --silent >/dev/null 2>&1; do
        i=$((i + 1)); [ "$i" -gt 60 ] && { echo "mariadb did not start, see /tmp/mariadb.log" >&2; exit 1; }
        sleep 1
    done
fi
mariadb -uroot <<'SQL'
CREATE DATABASE IF NOT EXISTS `openvas-tracker`;
CREATE USER IF NOT EXISTS 'otracker'@'localhost' IDENTIFIED BY 'otracker';
GRANT ALL PRIVILEGES ON `openvas-tracker`.* TO 'otracker'@'localhost';
SQL

if ! curl -fsS http://127.0.0.1:8080/api/health >/dev/null 2>&1; then
    # Run from a scratch directory so the Settings UI's .env writes stay out of /src.
    mkdir -p /tmp/openvas-tracker && cd /tmp/openvas-tracker
    OT_DATABASE_DSN='otracker:otracker@tcp(127.0.0.1:3306)/openvas-tracker?parseTime=true' \
    OT_JWT_SECRET=scanner-jwt-secret-0123456789abcdef0123456789 \
    OT_IMPORT_APIKEY=$IMPORT_APIKEY \
    OT_ADMIN_PASSWORD=$ADMIN_PASSWORD \
    OT_ENV_FILE=/tmp/openvas-tracker/env \
    nohup "$BIN" >/tmp/openvas-tracker.log 2>&1 &
    i=0
    until curl -fsS http://127.0.0.1:8080/api/health >/dev/null 2>&1; do
        i=$((i + 1)); [ "$i" -gt 60 ] && { echo "app did not start, see /tmp/openvas-tracker.log" >&2; exit 1; }
        sleep 1
    done
fi

# A local non-admin account (role viewer) for authorization testing; the app has no registration.
HASH=$(python3 -c 'import bcrypt,sys; print(bcrypt.hashpw(sys.argv[1].encode(), bcrypt.gensalt()).decode())' "$VIEWER_PASSWORD")
mariadb -uroot openvas-tracker -e "INSERT IGNORE INTO users (id, email, username, password, role)
    VALUES (UUID(), 'viewer@example.test', 'viewer', '$HASH', 'viewer')"

cat <<INFO
openvas-tracker is running on http://127.0.0.1:8080 (log: /tmp/openvas-tracker.log)
  admin login:   admin / $ADMIN_PASSWORD
  viewer login:  viewer / $VIEWER_PASSWORD
  import key:    $IMPORT_APIKEY   (header X-API-Key)
  sample import: curl -s -X POST -H "X-API-Key: $IMPORT_APIKEY" -H 'Content-Type: application/xml' \\
                   --data-binary @/src/testdata/openvas-sample-report.xml http://127.0.0.1:8080/api/import/openvas
  login:         curl -s -X POST -H 'Content-Type: application/json' \\
                   -d '{"username":"viewer","password":"$VIEWER_PASSWORD"}' http://127.0.0.1:8080/api/auth/login
INFO
