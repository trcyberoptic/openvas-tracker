package handler

import "testing"

func TestMaskEnvValues(t *testing.T) {
	in := map[string]string{
		"OT_JWT_SECRET":         "supersecretjwtkey12345",
		"OT_IMPORT_APIKEY":      "apikey-abcdefghijklmnop",
		"OT_ADMIN_PASSWORD":     "adminpw123",
		"OT_LDAP_BIND_PASSWORD": "ldappw1234",
		"OT_GMP_PASSWORD":       "gmppassword123",
		"OT_DATABASE_DSN":       "user:pw@tcp(localhost:3306)/db",
		"OT_SERVER_PORT":        "8080",
	}
	out := maskEnvValues(in)

	for _, k := range []string{"OT_JWT_SECRET", "OT_IMPORT_APIKEY", "OT_ADMIN_PASSWORD", "OT_LDAP_BIND_PASSWORD", "OT_GMP_PASSWORD", "OT_DATABASE_DSN"} {
		want := in[k][:4] + "********"
		if out[k] != want {
			t.Errorf("%s = %q, want masked %q", k, out[k], want)
		}
	}
	if out["OT_SERVER_PORT"] != "8080" {
		t.Errorf("OT_SERVER_PORT = %q, want passthrough", out["OT_SERVER_PORT"])
	}
}
