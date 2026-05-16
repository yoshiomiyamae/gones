package cheat

import (
	"fmt"
	"strings"
)

// genieCharset maps Game Genie letters to their 4-bit nibble values. The
// order is fixed by the Galoob hardware and matches every reference
// implementation (FCEUX, Mesen, Nestopia).
const genieCharset = "APZLGITYEOXUKSVN"

// DecodeGameGenie parses a 6- or 8-letter Game Genie code into a Cheat.
// 6-letter codes carry only address + replacement value; 8-letter codes
// additionally include a compare byte that gates the patch (the read is
// only replaced when the cartridge byte at addr equals Compare). The
// extra interlock lets multi-bank ROMs target a specific bank — without
// it, an unrelated bank with the same byte at that address would be
// patched too.
//
// The address bits are scrambled across the 24 / 32 input bits per
// Galoob's encoding. The bit-extraction expressions below are copied
// verbatim from the canonical algorithm — any rearrangement breaks
// existing community-published codes.
func DecodeGameGenie(code string) (Cheat, error) {
	code = strings.ToUpper(code)
	if len(code) != 6 && len(code) != 8 {
		return Cheat{}, fmt.Errorf("game genie: code length must be 6 or 8, got %d", len(code))
	}
	n := make([]byte, len(code))
	for i, c := range code {
		idx := strings.IndexRune(genieCharset, c)
		if idx < 0 {
			return Cheat{}, fmt.Errorf("game genie: invalid character %q in %q", c, code)
		}
		n[i] = byte(idx)
	}

	addr := uint16(0x8000) |
		(uint16(n[3]&7) << 12) |
		(uint16(n[5]&7) << 8) | (uint16(n[4]&8) << 8) |
		(uint16(n[2]&7) << 4) | (uint16(n[1]&8) << 4) |
		uint16(n[4]&7) | uint16(n[3]&8)

	c := Cheat{Address: addr, Source: code}

	if len(code) == 6 {
		c.Value = ((n[1] & 7) << 4) | ((n[0] & 8) << 4) | (n[0] & 7) | (n[5] & 8)
	} else {
		c.Value = ((n[1] & 7) << 4) | ((n[0] & 8) << 4) | (n[0] & 7) | (n[7] & 8)
		c.Compare = ((n[7] & 7) << 4) | ((n[6] & 8) << 4) | (n[6] & 7) | (n[5] & 8)
		c.HasCompare = true
	}
	return c, nil
}
