package cheat

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// LoadFile reads a .cht file: one cheat per line, blank lines and `#`
// comment lines ignored. Trailing `#` or `;` introduces an inline comment
// kept as the cheat's Comment label. Two formats are accepted:
//
//   - Game Genie: a 6- or 8-letter code (e.g. SXIOPO, GZEEAPNL)
//   - Raw poke:   AAAA:VV  or  AAAA:VV:CC  (hex address, value, compare)
//
// Returns the parsed cheats and the count of malformed lines (those are
// logged via the returned error chain but don't abort the load — a single
// bad code shouldn't lose the rest of the file).
func LoadFile(r io.Reader) ([]Cheat, error) {
	var cheats []Cheat
	var errs []string
	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line, comment := splitComment(scanner.Text())
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		c, err := parseLine(line)
		if err != nil {
			errs = append(errs, fmt.Sprintf("line %d: %v", lineNo, err))
			continue
		}
		c.Comment = comment
		cheats = append(cheats, c)
	}
	if err := scanner.Err(); err != nil {
		return cheats, err
	}
	if len(errs) > 0 {
		return cheats, fmt.Errorf("cheat file: %s", strings.Join(errs, "; "))
	}
	return cheats, nil
}

// splitComment returns (code, comment) split at the first `#` or `;`. The
// comment text has surrounding whitespace trimmed and the marker removed.
func splitComment(line string) (string, string) {
	for i, c := range line {
		if c == '#' || c == ';' {
			return line[:i], strings.TrimSpace(line[i+1:])
		}
	}
	return line, ""
}

func parseLine(line string) (Cheat, error) {
	if strings.Contains(line, ":") {
		return parseRaw(line)
	}
	return DecodeGameGenie(line)
}

// parseRaw handles `AAAA:VV` and `AAAA:VV:CC`. Hex throughout; no `$` or
// `0x` prefix required (but tolerated for friendliness).
func parseRaw(line string) (Cheat, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return Cheat{}, fmt.Errorf("raw cheat: want AAAA:VV[:CC], got %q", line)
	}
	addr, err := parseHex(parts[0], 16)
	if err != nil {
		return Cheat{}, fmt.Errorf("raw cheat: address: %w", err)
	}
	value, err := parseHex(parts[1], 8)
	if err != nil {
		return Cheat{}, fmt.Errorf("raw cheat: value: %w", err)
	}
	c := Cheat{Address: uint16(addr), Value: uint8(value), Source: line}
	if len(parts) == 3 {
		cmp, err := parseHex(parts[2], 8)
		if err != nil {
			return Cheat{}, fmt.Errorf("raw cheat: compare: %w", err)
		}
		c.Compare = uint8(cmp)
		c.HasCompare = true
	}
	return c, nil
}

func parseHex(s string, bits int) (uint64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return strconv.ParseUint(s, 16, bits)
}
