package util

func IsValidPort(port int) bool {
	return port > 0 && port <= 65536
}
