package network

import (
	"net"
	"time"
)

// CheckInternet checks if internet is reachable via DNS lookup
func CheckInternet(host string, timeout time.Duration) bool {
	if host == "" {
		host = "8.8.8.8"
	}

	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", host+":53")
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// Status returns a status string and color indicator
func Status() (string, bool) {
	if CheckInternet("8.8.8.8", 2*time.Second) {
		return "Online", true
	}
	return "Offline", false
}
