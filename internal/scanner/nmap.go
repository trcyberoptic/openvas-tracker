// internal/scanner/nmap.go
package scanner

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type NmapResult struct {
	Hosts []HostResult
}

type HostResult struct {
	Address  string
	Hostname string
	OS       string
	Status   string
	Ports    []PortResult
}

type PortResult struct {
	Port     int
	Protocol string
	State    string
	Service  string
	Product  string
	Version  string
}

// XML structures for nmap output parsing
type nmapRun struct {
	XMLName xml.Name   `xml:"nmaprun"`
	Hosts   []nmapHost `xml:"host"`
}

type nmapHost struct {
	Status    nmapStatus    `xml:"status"`
	Address   nmapAddress   `xml:"address"`
	Hostnames nmapHostnames `xml:"hostnames"`
	Ports     nmapPorts     `xml:"ports"`
	OS        nmapOS        `xml:"os"`
}

type nmapStatus struct {
	State string `xml:"state,attr"`
}

type nmapAddress struct {
	Addr string `xml:"addr,attr"`
}

type nmapHostnames struct {
	Names []nmapHostname `xml:"hostname"`
}

type nmapHostname struct {
	Name string `xml:"name,attr"`
}

type nmapPorts struct {
	Ports []nmapPort `xml:"port"`
}

type nmapPort struct {
	Protocol string      `xml:"protocol,attr"`
	PortID   int         `xml:"portid,attr"`
	State    nmapState   `xml:"state"`
	Service  nmapService `xml:"service"`
}

type nmapState struct {
	State string `xml:"state,attr"`
}

type nmapService struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
}

type nmapOS struct {
	Matches []nmapOSMatch `xml:"osmatch"`
}

type nmapOSMatch struct {
	Name     string `xml:"name,attr"`
	Accuracy string `xml:"accuracy,attr"`
}

func ParseNmapXML(r io.Reader) (*NmapResult, error) {
	var run nmapRun
	if err := xml.NewDecoder(r).Decode(&run); err != nil {
		return nil, fmt.Errorf("failed to parse nmap XML: %w", err)
	}

	result := &NmapResult{}
	for _, h := range run.Hosts {
		host := HostResult{
			Address: h.Address.Addr,
			Status:  h.Status.State,
		}
		if len(h.Hostnames.Names) > 0 {
			host.Hostname = h.Hostnames.Names[0].Name
		}
		if len(h.OS.Matches) > 0 {
			host.OS = h.OS.Matches[0].Name
		}
		for _, p := range h.Ports.Ports {
			host.Ports = append(host.Ports, PortResult{
				Port:     p.PortID,
				Protocol: p.Protocol,
				State:    p.State.State,
				Service:  p.Service.Name,
				Product:  p.Service.Product,
				Version:  p.Service.Version,
			})
		}
		result.Hosts = append(result.Hosts, host)
	}
	return result, nil
}

type NmapScanner struct {
	BinaryPath string
}

func NewNmapScanner(binaryPath string) *NmapScanner {
	return &NmapScanner{BinaryPath: binaryPath}
}

func (s *NmapScanner) Scan(ctx context.Context, target string, args ...string) (*NmapResult, error) {
	cmdArgs := append([]string{"-oX", "-", target}, args...)
	cmd := exec.CommandContext(ctx, s.BinaryPath, cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nmap execution failed: %w", err)
	}
	return ParseNmapXML(strings.NewReader(string(output)))
}
