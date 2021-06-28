package connect

import (
	"bufio"
	"bytes"
	"net"
	"os"
	"regexp"
	"strings"
)

var (
	cloudEx = `Version: .*(amazon)|Manufacturer: (Amazon)|Manufacturer: (Google)|Manufacturer: (Microsoft) Corporation`
	cloudRe = regexp.MustCompile(cloudEx)
)

func arch() (string, error) {
	output, err := execute([]string{"uname", "-i"}, false, nil)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func lscpu() (map[string]string, error) {
	output, err := execute([]string{"lscpu"}, false, nil)
	if err != nil {
		return nil, err
	}
	return lscpu2map(output), nil
}

func lscpu2map(b []byte) map[string]string {
	m := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		m[key] = val
	}
	return m
}

func cloudProvider() (string, error) {
	output, err := execute([]string{"dmidecode", "-t", "system"}, false, nil)
	if err != nil {
		return "", err
	}
	return findCloudProvider(output), nil
}

// findCloudProvider returns the cloud provider from "dmidecode -t system" output
func findCloudProvider(b []byte) string {
	match := cloudRe.FindSubmatch(b)
	for i, m := range match {
		if i != 0 && len(m) > 0 {
			return string(m)
		}
	}
	return ""
}

func hypervisor() (string, error) {
	output, err := execute([]string{"systemd-detect-virt", "-v"}, false, []int{0, 1})
	if err != nil {
		return "", err
	}
	if bytes.Equal(output, []byte("none")) {
		return "", nil
	}
	return string(output), nil
}

func uuid() (string, error) {
	if fileExists("/sys/hypervisor/uuid") {
		content, err := os.ReadFile("/sys/hypervisor/uuid")
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
	output, err := execute([]string{"dmidecode", "-s", "system-uuid"}, false, nil)
	if err != nil {
		return "", err
	}
	out := string(output)
	if strings.Contains(out, "Not Settable") || strings.Contains(out, "Not Present") {
		return "", nil
	}
	return out, nil
}

// getPrivateIPAddr returns the first private IP address on the host
func getPrivateIPAddr() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		ip, _, _ := net.ParseCIDR(addr.String())
		if privateIP(ip) {
			return ip.String(), nil
		}
	}
	return "", nil
}

// privateIP returns true if ip is in a RFC1918 range
func privateIP(ip net.IP) bool {
	for _, block := range []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"} {
		_, ipNet, _ := net.ParseCIDR(block)
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

func hostname() string {
	name, err := os.Hostname()
	// TODO the Ruby version has this "(none)" check - why?
	if err == nil && name != "" && name != "(none)" {
		return name
	}
	Debug.Print(err)
	ip, err := getPrivateIPAddr()
	if err != nil {
		Debug.Print(err)
		return ""
	}
	return ip
}

// readValues calls read_values from SUSE/s390-tools
func readValues(arg string) ([]byte, error) {
	output, err := execute([]string{"read_values", arg}, false, nil)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// readValues2map parses the output of "read_values -s" on s390
func readValues2map(b []byte) map[string]string {
	br := bufio.NewScanner(bytes.NewReader(b))
	m := make(map[string]string)
	for br.Scan() {
		line := br.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		m[key] = val
	}
	return m
}
