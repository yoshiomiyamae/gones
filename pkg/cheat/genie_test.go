package cheat

import "testing"

// TestDecodeGameGenieKnownCodes verifies decoding against widely-published
// codes whose decoded (address, value) pairs are documented in multiple
// references. Failing any case here means the bit shuffling is wrong;
// updating the algorithm is the fix, not relaxing the expected values.
func TestDecodeGameGenieKnownCodes(t *testing.T) {
	cases := []struct {
		code    string
		addr    uint16
		value   uint8
		compare uint8
		hasCmp  bool
	}{
		// Identity case: all-A code (all nibbles zero) must produce
		// $8000:$00 — the address gets the implicit 0x8000 high bit.
		{code: "AAAAAA", addr: 0x8000, value: 0x00},
		// SMB infinite lives (Galoob-published, classic code).
		{code: "SXIOPO", addr: 0x91D9, value: 0xAD},
		// 8-letter compare-gated code — exercises the compare-byte path.
		{code: "YEKPSPSI", addr: 0x99C5, value: 0x07, compare: 0xD5, hasCmp: true},
	}
	for _, tc := range cases {
		c, err := DecodeGameGenie(tc.code)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.code, err)
			continue
		}
		if c.Address != tc.addr {
			t.Errorf("%s: addr = %#04x, want %#04x", tc.code, c.Address, tc.addr)
		}
		if c.Value != tc.value {
			t.Errorf("%s: value = %#02x, want %#02x", tc.code, c.Value, tc.value)
		}
		if c.HasCompare != tc.hasCmp {
			t.Errorf("%s: hasCompare = %v, want %v", tc.code, c.HasCompare, tc.hasCmp)
		}
		if tc.hasCmp && c.Compare != tc.compare {
			t.Errorf("%s: compare = %#02x, want %#02x", tc.code, c.Compare, tc.compare)
		}
	}
}

func TestDecodeGameGenieInvalid(t *testing.T) {
	cases := []string{
		"",
		"ABC",     // wrong length
		"ABCDEFG", // 7 chars
		"SXIOP@",  // bad char
	}
	for _, code := range cases {
		if _, err := DecodeGameGenie(code); err == nil {
			t.Errorf("%q: expected error, got nil", code)
		}
	}
}

func TestApply(t *testing.T) {
	m := NewManager()
	m.Add(Cheat{Address: 0x8001, Value: 0x42})
	m.Add(Cheat{Address: 0x8002, Value: 0x99, Compare: 0x55, HasCompare: true})

	if got := m.Apply(0x8001, 0x00); got != 0x42 {
		t.Errorf("unconditional patch: got %#02x, want 0x42", got)
	}
	if got := m.Apply(0x8002, 0x55); got != 0x99 {
		t.Errorf("compare match: got %#02x, want 0x99", got)
	}
	if got := m.Apply(0x8002, 0x44); got != 0x44 {
		t.Errorf("compare miss: got %#02x, want passthrough 0x44", got)
	}
	if got := m.Apply(0x9000, 0x33); got != 0x33 {
		t.Errorf("no cheat: got %#02x, want passthrough 0x33", got)
	}

	m.ToggleAll()
	if got := m.Apply(0x8001, 0x00); got != 0x00 {
		t.Errorf("disabled: got %#02x, want passthrough 0x00", got)
	}
}
