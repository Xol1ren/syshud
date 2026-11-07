package sysinfo

// ParseUint parses an ASCII decimal integer from b starting at offset 0,
// skipping any leading spaces or tabs.
// Returns the parsed integer and the number of bytes consumed.
// Zero heap allocations.
func ParseUint(b []byte) (uint64, int) {
	var n uint64
	var i int
	for i < len(b) && (b[i] == ' ' || b[i] == '\t') {
		i++
	}
	start := i
	for i < len(b) && b[i] >= '0' && b[i] <= '9' {
		n = n*10 + uint64(b[i]-'0')
		i++
	}
	if i == start {
		return 0, i
	}
	return n, i
}

// ParseInt parses a signed ASCII decimal integer from b.
// Zero heap allocations.
func ParseInt(b []byte) (int64, int) {
	var i int
	for i < len(b) && (b[i] == ' ' || b[i] == '\t') {
		i++
	}
	if i >= len(b) {
		return 0, i
	}
	neg := false
	if b[i] == '-' {
		neg = true
		i++
	} else if b[i] == '+' {
		i++
	}
	var n int64
	start := i
	for i < len(b) && b[i] >= '0' && b[i] <= '9' {
		n = n*10 + int64(b[i]-'0')
		i++
	}
	if i == start {
		return 0, i
	}
	if neg {
		n = -n
	}
	return n, i
}
