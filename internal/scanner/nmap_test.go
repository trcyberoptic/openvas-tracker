// internal/scanner/nmap_test.go
package scanner

import (
	"strings"
	"testing"
)

func TestParseNmapXML(t *testing.T) {
	xml := `<?xml version="1.0"?>
<nmaprun scanner="nmap" args="nmap -sV 192.168.1.1" start="1234567890">
  <host starttime="1234567890" endtime="1234567899">
    <status state="up"/>
    <address addr="192.168.1.1" addrtype="ipv4"/>
    <hostnames><hostname name="router.local" type="PTR"/></hostnames>
    <ports>
      <port protocol="tcp" portid="22">
        <state state="open"/>
        <service name="ssh" product="OpenSSH" version="8.9"/>
      </port>
      <port protocol="tcp" portid="80">
        <state state="open"/>
        <service name="http" product="nginx" version="1.18.0"/>
      </port>
    </ports>
    <os><osmatch name="Linux 5.x" accuracy="95"/></os>
  </host>
</nmaprun>`

	result, err := ParseNmapXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNmapXML error: %v", err)
	}
	if len(result.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(result.Hosts))
	}
	host := result.Hosts[0]
	if host.Address != "192.168.1.1" {
		t.Errorf("expected address 192.168.1.1, got %s", host.Address)
	}
	if len(host.Ports) != 2 {
		t.Errorf("expected 2 ports, got %d", len(host.Ports))
	}
	if host.Ports[0].Service != "ssh" {
		t.Errorf("expected service ssh, got %s", host.Ports[0].Service)
	}
	if host.OS != "Linux 5.x" {
		t.Errorf("expected OS Linux 5.x, got %s", host.OS)
	}
}
