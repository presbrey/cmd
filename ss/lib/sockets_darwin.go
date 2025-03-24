package lib

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// GetSockets retrieves socket information on macOS using lsof
func GetSockets(tcp, udp, listeningOnly, all bool) ([]Socket, error) {
	var sockets []Socket

	// Build lsof command arguments
	args := []string{"-nP", "-i"} // -n for numeric, -P for numeric ports
	if tcp && !udp {
		args = append(args, "tcp")
	} else if udp && !tcp {
		args = append(args, "udp")
	}

	// Execute lsof command
	cmd := exec.Command("lsof", args...)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "COMMAND") {
		return nil, fmt.Errorf("error executing lsof: %v", err)
	}

	// Parse lsof output
	lines := strings.Split(string(output), "\n")

	// Regular expressions for parsing
	ipv4PortRegex := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+|\*):(\d+|\*)`)
	ipv6PortRegex := regexp.MustCompile(`\[([0-9a-fA-F:]+|\*)\]:(\d+|\*)`)

	// Skip header
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		// Extract basic information
		procName := fields[0]
		pid, _ := strconv.Atoi(fields[1])

		// Determine protocol
		proto := ""
		if strings.Contains(fields[7], "TCP") {
			proto = "tcp"
			// Skip if only UDP is requested
			if !tcp {
				continue
			}
		} else if strings.Contains(fields[7], "UDP") {
			proto = "udp"
			// Skip if only TCP is requested
			if !udp {
				continue
			}
		} else {
			continue
		}

		// Extract state
		state := ""
		if len(fields) >= 10 {
			state = fields[9]
			state = strings.Trim(state, "()")
		} else if proto == "udp" {
			state = "UNCONN"
		} else {
			state = "UNKNOWN"
		}

		// Skip non-listening sockets if listening only is requested
		if listeningOnly && state != "LISTEN" && !all {
			continue
		}

		// Parse address field (field 8)
		addrField := fields[8]
		var localAddr, remoteAddr string
		var localPort, remotePort int

		if strings.Contains(addrField, "->") {
			// Connected socket
			parts := strings.Split(addrField, "->")
			localAddr, localPort = ParseAddrPort(parts[0], ipv4PortRegex, ipv6PortRegex)
			remoteAddr, remotePort = ParseAddrPort(parts[1], ipv4PortRegex, ipv6PortRegex)
		} else {
			// Listening or unconnected socket
			localAddr, localPort = ParseAddrPort(addrField, ipv4PortRegex, ipv6PortRegex)
		}

		// Create socket object
		socket := Socket{
			Netid:       proto,
			State:       state,
			LocalAddr:   localAddr,
			LocalPort:   localPort,
			RemoteAddr:  remoteAddr,
			RemotePort:  remotePort,
			ProcessName: procName,
			PID:         pid,
		}

		sockets = append(sockets, socket)
	}

	return sockets, nil
}

// ParseAddrPort parses an address:port string and returns them separately
func ParseAddrPort(addrPort string, ipv4Regex, ipv6Regex *regexp.Regexp) (string, int) {
	// Try IPv4 format first
	matches := ipv4Regex.FindStringSubmatch(addrPort)
	if len(matches) >= 3 {
		port := 0
		if matches[2] != "*" {
			port, _ = strconv.Atoi(matches[2])
		}
		return matches[1], port
	}

	// Try IPv6 format
	matches = ipv6Regex.FindStringSubmatch(addrPort)
	if len(matches) >= 3 {
		port := 0
		if matches[2] != "*" {
			port, _ = strconv.Atoi(matches[2])
		}
		return matches[1], port
	}

	return "*", 0
}
