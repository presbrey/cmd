package lib

// Socket represents a network socket
type Socket struct {
	Netid       string // Protocol (tcp, udp)
	State       string // Socket state (LISTEN, ESTABLISHED, etc.)
	LocalAddr   string // Local address
	LocalPort   int    // Local port
	RemoteAddr  string // Remote address
	RemotePort  int    // Remote port
	ProcessName string // Process name
	PID         int    // Process ID
}
