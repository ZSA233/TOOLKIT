package proxytest

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"
)

func readUntil(conn net.Conn, marker []byte, limit int) ([]byte, error) {
	reader := bufio.NewReader(conn)
	buffer := make([]byte, 0, 4096)
	for {
		chunk, err := reader.ReadBytes(marker[len(marker)-1])
		buffer = append(buffer, chunk...)
		if len(buffer) > limit {
			return nil, fmt.Errorf("response header too large")
		}
		if hasSuffixBytes(buffer, marker) || containsBytes(buffer, marker) {
			return buffer, nil
		}
		if err != nil {
			if err == io.EOF {
				return buffer, nil
			}
			return nil, err
		}
	}
}

func tlsClient(conn net.Conn, host string) *tls.Conn {
	return tls.Client(conn, &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	})
}

func secondsSince(start time.Time) float64 {
	return time.Since(start).Seconds()
}

func containsBytes(haystack []byte, needle []byte) bool {
	for index := 0; index+len(needle) <= len(haystack); index++ {
		if hasSuffixBytes(haystack[index:index+len(needle)], needle) {
			return true
		}
	}
	return false
}

func hasSuffixBytes(value []byte, suffix []byte) bool {
	if len(value) < len(suffix) {
		return false
	}
	start := len(value) - len(suffix)
	for index := range suffix {
		if value[start+index] != suffix[index] {
			return false
		}
	}
	return true
}
