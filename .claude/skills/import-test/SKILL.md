---
name: import-test
description: Import test data on scanner01 and verify dashboard shows results
---

Test the import pipeline on the production server:

1. Get the API key:
   ```bash
   API_KEY=$(ssh scanner01 "grep OT_IMPORT_APIKEY /etc/openvas-tracker/env | cut -d= -f2")
   ```

2. Import sample report:
   ```bash
   scp testdata/openvas-sample-report.xml scanner01:/tmp/
   ssh scanner01 "curl -s -X POST http://localhost:8080/api/import/openvas -H 'X-API-Key: $API_KEY' -H 'Content-Type: application/xml' --data-binary @/tmp/openvas-sample-report.xml"
   ```

3. Verify the response shows `vulnerabilities_imported > 0`.

4. Check dashboard:
   ```bash
   TOKEN=$(ssh scanner01 "curl -s -X POST http://localhost:8080/api/auth/login -H 'Content-Type: application/json' -d '{\"email\":\"admin@openvas-tracker.local\",\"password\":\"changeme123\"}'" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
   ssh scanner01 "curl -s http://localhost:8080/api/dashboard -H 'Authorization: Bearer $TOKEN'"
   ```

5. Report results to the user.
