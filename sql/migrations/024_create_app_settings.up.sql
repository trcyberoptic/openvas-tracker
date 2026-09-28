-- Settings editable at runtime from the web UI (mail relay, notification
-- recipients). Read on every use, so changes apply without a restart.
CREATE TABLE app_settings (
    name       VARCHAR(64) NOT NULL PRIMARY KEY,
    value      TEXT        NOT NULL,
    updated_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
