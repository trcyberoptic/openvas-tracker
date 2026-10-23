# Changelog

User-facing changes per release. The release workflow publishes the section of the tagged version as the GitHub release notes and fails if it is missing.

## v2.4.1 - 2026-10-23

### Fixed
- **Scans that ran no vulnerability tests are no longer imported.** When the Greenbone scanner fails to load its tests (seen on 2026-10-22 after a restart of `ospd-openvas`), GVM still finishes the scan as "Done", but the report holds nothing except one "Hostname Determination Reporting" result per host. The tracker imported such a report like a normal scan, so every open ticket on the scanned hosts counted as "not found any more" and moved to `pending_resolution` - after three such scans they would have been closed as fixed. The tracker now rejects a report whose results all come from a single test (HTTP 400, logged as `import rejected: ...`). The fetch script skips it with a warning and continues with the other reports.

### Known limits
- A report that was filtered down to a single test before a manual upload is rejected as well.

## v2.4.0 - 2026-10-19

### Changed
- **Reports from all scan tasks are imported.** Every GVM alert now imports each task's latest finished report that the tracker doesn't have yet, oldest first. Before, an alert imported only the report with the newest start time, so with several tasks running at the same time a report could be skipped without notice ([#52](https://github.com/trcyberoptic/openvas-tracker/issues/52)).
- **No double imports.** The tracker stores the GVM report ID with each scan and answers a repeated upload of the same report with `409 Conflict`.
- **Outdated reports are skipped.** A report that ended before the newest imported one is not imported, so old results can't reopen tickets that newer scans already closed.
- Greenbone feed versions are now updated on every alert, also when there was no new report.

### Upgrade
- Database migration 026 runs automatically on startup.
- The fetch script is not part of the .deb. After installing the package, copy `deploy/openvas-tracker-fetch-latest` to `/usr/local/bin/` (`deploy/install.sh` does this). The new script needs this tracker version, so update the tracker first.

### Known limits
- Only each task's last finished report is considered. If a task finishes again before its previous report was imported, the older report is skipped.
