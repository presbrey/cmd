package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/presbrey/cmd/ss/lib"
)

func main() {
	// Define flags but don't use the flag package for parsing
	var numeric, listening, process, tcp, udp, all, help bool

	// Custom usage
	usage := func() {
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Println("Options:")
		fmt.Println("  -a\tDisplay all sockets (listening and non-listening)")
		fmt.Println("  -h\tDisplay help")
		fmt.Println("  -l\tDisplay only listening sockets")
		fmt.Println("  -n\tShow numeric addresses instead of resolving host names")
		fmt.Println("  -p\tShow process using socket")
		fmt.Println("  -t\tDisplay TCP sockets")
		fmt.Println("  -u\tDisplay UDP sockets")
		fmt.Println("\nExamples:")
		fmt.Println("  ss -t       # Show TCP sockets")
		fmt.Println("  ss -ua      # Show all UDP sockets")
		fmt.Println("  ss -nlpt    # Show listening TCP socket processes in numeric format")
	}

	// Parse command line arguments manually to support combined flags
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
			fmt.Fprintf(os.Stderr, "Unknown argument: %s\n", arg)
			usage()
			os.Exit(1)
		}

		// Process each character in the flag
		for _, c := range arg[1:] {
			switch c {
			case 'n':
				numeric = true
			case 'l':
				listening = true
			case 'p':
				process = true
			case 't':
				tcp = true
			case 'u':
				udp = true
			case 'a':
				all = true
			case 'h':
				usage()
				os.Exit(0)
			default:
				fmt.Fprintf(os.Stderr, "Unknown flag: -%c\n", c)
				usage()
				os.Exit(1)
			}
		}
	}

	// Check for help
	if help {
		usage()
		os.Exit(0)
	}

	// If neither TCP nor UDP is specified, default to TCP
	if !tcp && !udp {
		tcp = true
	}

	// Display socket information using range function
	displaySocketsWithRange(tcp, udp, listening, all, numeric, process)
}

// getSockets retrieves socket information based on the specified filters
// Platform-specific implementation is in sockets_*.go files

// displaySocketsWithRange uses the range function to display sockets
func displaySocketsWithRange(tcp, udp, listening, all, numeric, showProcess bool) {
	// Print header in the style of the actual ss command
	fmt.Printf("%-5s %-11s %-23s %-23s", "Netid", "State", "Local Address:Port", "Peer Address:Port")
	if showProcess {
		fmt.Printf(" %-20s", "Process")
	}
	fmt.Println()

	// Use range function to process each socket
	for s := range lib.Sockets(tcp, udp, listening, all) {
		localAddr := s.LocalAddr
		remoteAddr := s.RemoteAddr

		// Resolve addresses if not numeric
		if !numeric {
			if localAddr != "*" && net.ParseIP(localAddr) != nil {
				names, err := net.LookupAddr(localAddr)
				if err == nil && len(names) > 0 {
					localAddr = strings.TrimSuffix(names[0], ".")
				}
			}

			if remoteAddr != "" && remoteAddr != "*" && net.ParseIP(remoteAddr) != nil {
				names, err := net.LookupAddr(remoteAddr)
				if err == nil && len(names) > 0 {
					remoteAddr = strings.TrimSuffix(names[0], ".")
				}
			}
		}

		// Format addresses with ports
		localAddrPort := formatAddrPort(localAddr, s.LocalPort)
		remoteAddrPort := "*:*"
		if remoteAddr != "" {
			remoteAddrPort = formatAddrPort(remoteAddr, s.RemotePort)
		}

		// Print socket information
		fmt.Printf("%-5s %-11s %-23s %-23s", s.Netid, s.State, localAddrPort, remoteAddrPort)

		// Print process information if requested
		if showProcess {
			fmt.Printf(" %-20s", fmt.Sprintf("%s(%d)", s.ProcessName, s.PID))
		}

		fmt.Println()
	}
}

func formatAddrPort(addr string, port int) string {
	// Format IPv6 addresses properly
	if strings.Contains(addr, ":") && addr != "*" {
		return fmt.Sprintf("[%s]:%d", addr, port)
	}
	return fmt.Sprintf("%s:%d", addr, port)
}
