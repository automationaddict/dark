package network

import (
	"encoding/binary"
	"net"
	"os"
	"strconv"
	"strings"
)

// --- low level helpers ---

func readSysfsString(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readSysfsInt(path string) int {
	s := readSysfsString(path)
	if s == "" {
		return -1
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return n
}

func readSysfsUint(path string) uint64 {
	s := readSysfsString(path)
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// parseHexLE32 reads an 8-character little-endian hex string as a
// uint32. /proc/net/route stores IPv4 addresses this way.
func parseHexLE32(s string) uint32 {
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0
	}
	// Bytes are in network order in the integer, but the file writes
	// them in host order — flip them.
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n))
	return binary.LittleEndian.Uint32(b)
}

func uint32ToIP(n uint32) net.IP {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return net.IP(b)
}

func bitsInMask(mask uint32) int {
	count := 0
	for mask != 0 {
		count += int(mask & 1)
		mask >>= 1
	}
	return count
}

// parseHexIPv6 decodes a 32-character hex string into a net.IP.
// /proc/net/ipv6_route uses this format with no separators.
func parseHexIPv6(s string) (net.IP, bool) {
	if len(s) != 32 {
		return nil, false
	}
	out := make(net.IP, 16)
	for i := 0; i < 16; i++ {
		b, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return nil, false
		}
		out[i] = byte(b)
	}
	return out, true
}
