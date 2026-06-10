package cleanup

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// sizeRe matches a human size like "503.9MB", "3.4 GB", "800kB" or "12GiB".
var sizeRe = regexp.MustCompile(`(?i)([\d.]+)\s*([KMGTP]?i?B)`)

var unitBytes = map[string]float64{
	"B":  1,
	"KB": 1e3, "KIB": 1 << 10,
	"MB": 1e6, "MIB": 1 << 20,
	"GB": 1e9, "GIB": 1 << 30,
	"TB": 1e12, "TIB": 1 << 40,
	"PB": 1e15, "PIB": 1 << 50,
}

// parseSize converts the first size token in s to bytes, reporting ok=false
// when no recognizable size is present.
func parseSize(s string) (float64, bool) {
	m := sizeRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	mult, ok := unitBytes[strings.ToUpper(m[2])]
	if !ok {
		return 0, false
	}
	return n * mult, true
}

// formatBytes renders a byte count as a compact human string (e.g. "1.2GB").
func formatBytes(b float64) string {
	const k = 1000.0
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	for b >= k && i < len(units)-1 {
		b /= k
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f%s", b, units[i])
	}
	return fmt.Sprintf("%.1f%s", b, units[i])
}

// formatKB renders a kilobyte count (as from `du -sk`) as a human string.
func formatKB(kb float64) string {
	return formatBytes(kb * 1024)
}
