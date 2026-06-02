package gui

import (
	"os"
	"strings"
	"time"
)

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func splitLines(data []byte) []string {
	s := strings.TrimRight(string(data), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func now() time.Time {
	return time.Now()
}

func elapsed(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
