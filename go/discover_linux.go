package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const tcpStateListen = "0A" // /proc/net/tcp st field value for LISTEN

// discoverPort returns the first TCP LISTEN port owned by pid, or ok=false.
func discoverPort(pid int) (int, bool) {
	inodes := socketInodes(pid)
	if len(inodes) == 0 {
		return 0, false
	}
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if port, ok := scanListen(path, inodes); ok {
			return port, true
		}
	}
	return 0, false
}

// socketInodes returns the set of socket inodes held open by pid.
func socketInodes(pid int) map[string]bool {
	inodes := map[string]bool{}
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return inodes
	}
	for _, e := range entries {
		link, err := os.Readlink(fdDir + "/" + e.Name())
		if err != nil {
			continue
		}
		if strings.HasPrefix(link, "socket:[") && strings.HasSuffix(link, "]") {
			inodes[link[len("socket:["):len(link)-1]] = true
		}
	}
	return inodes
}

// scanListen reads a /proc/net/tcp{,6} file and returns the port of the first
// LISTEN row whose inode is in the given set.
func scanListen(path string, inodes map[string]bool) (int, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Scan() // skip header
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 10 {
			continue
		}
		if fields[3] != tcpStateListen {
			continue
		}
		if !inodes[fields[9]] {
			continue
		}
		// fields[1] is local_address "HEXIP:HEXPORT"
		parts := strings.Split(fields[1], ":")
		if len(parts) != 2 {
			continue
		}
		port, err := strconv.ParseInt(parts[1], 16, 32)
		if err != nil {
			continue
		}
		return int(port), true
	}
	return 0, false
}
