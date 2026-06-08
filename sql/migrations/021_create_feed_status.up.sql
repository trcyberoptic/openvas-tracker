-- sql/migrations/021_create_feed_status.up.sql
-- Latest observed Greenbone feed versions (NVT/SCAP/CERT/GVMD_DATA) so the UI
-- can show that the GVM feed auto-update is current.
CREATE TABLE feed_status (
    feed_type    VARCHAR(32)  NOT NULL PRIMARY KEY,
    feed_name    VARCHAR(255) NOT NULL,
    version      VARCHAR(32)  NOT NULL,
    first_seen   DATETIME     NOT NULL,
    last_seen    DATETIME     NOT NULL,
    last_changed DATETIME     NOT NULL
);
