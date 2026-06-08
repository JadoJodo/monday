package custom

import "strings"

// nonEmptyLines splits the given output blocks into trimmed, non-empty lines.
func nonEmptyLines(blocks ...string) []string {
	var lines []string
	for _, block := range blocks {
		for ln := range strings.SplitSeq(block, "\n") {
			if strings.TrimSpace(ln) != "" {
				lines = append(lines, strings.TrimRight(ln, "\r "))
			}
		}
	}
	return lines
}
