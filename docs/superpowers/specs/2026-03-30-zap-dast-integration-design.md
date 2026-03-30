# ZAP DAST Integration Design

## Zusammenfassung

Integration von OWASP ZAP als zweiten Scanner-Typ in den OpenVAS-Tracker. ZAP-Reports werden per Webhook importiert (JSON Traditional Report), durchlaufen den gleichen Ticket-Lifecycle wie OpenVAS-Findings, mit URL-granularem Fingerprinting für Web-Applikations-Findings.

## Anforderungen

- **Aktives + passives DAST** via OWASP ZAP
- **Import-only** — Scans werden extern in ZAP gestartet, Ergebnisse per API-Webhook gepostet
- **URL-granulares Ticketing** — jede betroffene URL+Parameter-Kombination bekommt ein eigenes Ticket
- **Interne und externe Applikationen** (Pentests, Audits, eigene Apps)
- **Gleicher Ticket-Lifecycle** — Auto-Resolve, Flapping, Risk-Accept-Rules funktionieren für ZAP-Findings

## Scanner-Abstraktion

### Finding-Struct

Generalisierter Datentyp, der sowohl OpenVAS- als auch ZAP-Findings abbildet:

```go
// internal/scanner/scanner.go
type Finding struct {
    Host        string   // IP oder Hostname
    Hostname    string   // Reverse-DNS / App-Name
    Port        string   // z.B. "443"
    Protocol    string   // tcp/udp
    URL         string   // Voller URL-Pfad (leer bei Netzwerk-Scans)
    Parameter   string   // Betroffener Parameter (leer bei Netzwerk-Scans)
    Title       string
    Description string
    Severity    string   // critical/high/medium/low/info
    CVSSScore   float64
    CVEID       string   // Leer bei DAST-Findings
    CWEID       string   // z.B. "79" für XSS
    OID         string
    Solution    string
    Evidence    string   // Proof-Snippet aus dem Response
    Confidence  string   // high/medium/low/confirmed
    ScanType    string   // "openvas" oder "zap"
}
```

### Parser-Interface

```go
type Parser interface {
    Parse(r io.Reader) ([]Finding, error)
    ScanType() string
}
```

- `OpenVASParser` — Refactoring von `ParseOpenVASXML()` auf das Interface, gibt `[]Finding` zurück
- `ZAPParser` — Neuer Parser für ZAP JSON Traditional Report

## ZAP JSON Parser

### Input-Format

ZAP Traditional JSON Report Struktur:

```json
{
  "site": [{
    "host": "example.com",
    "port": "443",
    "ssl": "true",
    "alerts": [{
      "pluginid": "40012",
      "alert": "Cross Site Scripting (Reflected)",
      "riskcode": "3",
      "confidence": "2",
      "cweid": "79",
      "desc": "...",
      "solution": "...",
      "instances": [{
        "uri": "https://example.com/app/search",
        "method": "GET",
        "param": "q",
        "attack": "<script>alert(1)</script>",
        "evidence": "<script>alert(1)</script>"
      }]
    }]
  }]
}
```

### Mapping-Regeln

| ZAP-Feld | Finding-Feld | Transformation |
|----------|-------------|----------------|
| `site.host` | `Host` | Direkt |
| `site.port` | `Port` | Direkt |
| `alert.riskcode` | `Severity` | 3=high, 2=medium, 1=low, 0=info |
| `alert.riskcode` | `CVSSScore` | 3→7.0, 2→4.0, 1→2.0, 0→0.0 |
| `alert.confidence` | `Confidence` | 4=confirmed, 3=high, 2=medium, 1=low |
| `alert.cweid` | `CWEID` | Direkt |
| `alert.alert` | `Title` | Direkt |
| `alert.desc` | `Description` | HTML-Tags strippen |
| `alert.solution` | `Solution` | HTML-Tags strippen |
| `instance.uri` | `URL` | Pfad-Teil extrahiert (ohne Host) |
| `instance.param` | `Parameter` | Direkt |
| `instance.evidence` | `Evidence` | Direkt |

- Jede `instance` innerhalb eines `alert` wird zu einem eigenen `Finding`
- ZAP kennt kein "critical" — Default-CVSS-Werte ermöglichen korrekte Sortierung
- Info-Level Findings mit CVSS 0.0 werden übersprungen (analog zu OpenVAS)

## Fingerprinting

### Netzwerk-Findings (OpenVAS)

Unverändert:

- CVE vorhanden: `fingerprint = cve_id`
- Kein CVE: `fingerprint = "title:" + title`
- Schlüssel: `(affected_host, fingerprint)`

### Web-Findings (ZAP)

URL-granular:

- CWE vorhanden: `fingerprint = "cwe:" + cweid + ":url:" + urlPath + ":param:" + param`
- Kein CWE: `fingerprint = "title:" + title + ":url:" + urlPath`
- Schlüssel: `(affected_host, fingerprint)`

Dies stellt sicher, dass ein XSS auf `/app/search?q=` und ein XSS auf `/app/comment?text=` separate Tickets erzeugen.

## Datenbank-Änderungen

### Migration 020: ZAP-Felder

```sql
-- 020_add_zap_fields.up.sql
ALTER TABLE vulnerabilities
  ADD COLUMN url VARCHAR(2048) DEFAULT '' AFTER hostname,
  ADD COLUMN parameter VARCHAR(255) DEFAULT '' AFTER url,
  ADD COLUMN evidence TEXT AFTER solution,
  ADD COLUMN confidence VARCHAR(20) DEFAULT '' AFTER evidence,
  ADD COLUMN cwe_id VARCHAR(20) DEFAULT '' AFTER cve_id;

ALTER TABLE scans MODIFY COLUMN scan_type ENUM('nmap', 'openvas', 'zap', 'custom');

-- 020_add_zap_fields.down.sql
ALTER TABLE vulnerabilities
  DROP COLUMN url,
  DROP COLUMN parameter,
  DROP COLUMN evidence,
  DROP COLUMN confidence,
  DROP COLUMN cwe_id;

ALTER TABLE scans MODIFY COLUMN scan_type ENUM('nmap', 'openvas', 'custom');
```

Keine neuen Tabellen. Alle neuen Spalten haben Defaults, kein Breaking Change für bestehende Daten.

## API-Endpoint

### POST /api/import/zap

- **Auth:** API-Key (`OT_IMPORT_APIKEY`), gleicher Key wie OpenVAS
- **Body:** ZAP Traditional JSON Report
- **Body Limit:** 50MB (gleich wie OpenVAS-Import, via Skipper)
- **Response:** Identisches `ImportResult`-Format

```json
{
  "scan_id": "uuid",
  "vulnerabilities_imported": 42,
  "vulnerabilities_skipped": 3,
  "tickets_created": 15,
  "tickets_reopened": 2,
  "tickets_auto_resolved": 1
}
```

### Import-Flow

1. Handler empfängt JSON, ruft `scanner.ParseZAPJSON()` auf
2. Parser liefert `[]Finding`
3. `ImportService.Import()` wird mit `[]Finding` und `scanType="zap"` aufgerufen
4. Gleicher Code-Pfad: Fingerprinting, Ticket-Erstellung, Auto-Resolve, Risk-Rules

### Kein GET-Endpoint

ZAP hat kein GMP-Socket-Äquivalent. Kein Fetch-Trigger nötig.

## Auto-Resolve & Flapping

### Scope-Trennung

- `scan_hosts` wird auch für ZAP-Scans befüllt — mit `host:port` der gescannten Site(s)
- Auto-Resolve ist geschränkt auf `(host, scan_type)`: Ein ZAP-Scan resolved nur ZAP-Tickets für denselben Host
- Kein Cross-Scanner-Resolve: OpenVAS-Scan resolved keine ZAP-Tickets und umgekehrt

### Flapping-Mechanismus

Identisch zu OpenVAS:

- Finding fehlt → `consecutive_misses++`
- Erster Miss: `open` → `pending_resolution`
- Nach `OT_AUTORESOLVE_THRESHOLD` Misses: `pending_resolution` → `fixed`
- Finding taucht wieder auf: Counter reset, Ticket zurück auf `open`

### Kein URL-Pfad-basiertes Scoping

URL-Pfade sind zu instabil (Deployments, Versionsänderungen). Scope basiert auf Host-Ebene. Wenn ein ZAP-Scan `app1.example.com:443` enthält, werden alle offenen ZAP-Tickets für diesen Host geprüft.

## Frontend-Änderungen

### Scan-Liste

- Scan-Type-Badge: `OpenVAS` (grün) / `ZAP` (blau)
- Filter-Option nach Scan-Type

### Vulnerability-Detail / Ticket-Detail

Bedingte Anzeige neuer Felder (nur wenn Wert vorhanden):

| Feld | Darstellung |
|------|-------------|
| URL | Klickbarer Pfad |
| Parameter | Inline-Text |
| Evidence | Code-Block (monospace, max 500 Zeichen) |
| CWE ID | Badge mit Link zu `cwe.mitre.org/data/definitions/{id}` |
| Confidence | Badge: confirmed=grün, high=blau, medium=gelb, low=grau |

### Ticket-Liste

- Neue optionale Spalte "Source" (scan_type) — per Column-Toggle, nicht default-sichtbar
- Bestehende CVSS-Sortierung funktioniert weiterhin

### Keine neuen Navigations-Einträge

ZAP-Findings erscheinen in den gleichen Views (Tickets, Scans, Vulnerabilities). Trennung über Filter, nicht separate Seiten.

## Risk-Accept-Rules

Bestehender Mechanismus funktioniert für ZAP-Findings:

- **Fingerprint-Match:** Rule mit `fingerprint = "cwe:79:url:/app/search:param:q"` matcht exakt dieses Finding
- **Host-Pattern:** `*` oder spezifische IP — funktioniert wie bei OpenVAS
- **Titel-basierte Rules:** Rule mit `fingerprint = "title:Cross Site Scripting (Reflected)"` matcht alle XSS-Findings unabhängig von URL

Keine Änderungen am Rule-Mechanismus nötig.

## Nicht im Scope

- **Scan-Steuerung** — kein Start/Stop/Status von ZAP-Scans aus dem Tracker
- **ZAP API-Anbindung** — kein direkter ZAP-API-Zugriff
- **Nuclei oder andere Scanner** — Architektur vorbereitet, aber nicht implementiert
- **CVSS-Lookup für CWEs** — Default-Werte pro Severity-Level reichen aus
- **Separate ZAP-spezifische Views** — alles in bestehenden Views mit Filtern
