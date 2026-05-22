package cheat

import (
	"strings"
	"testing"
)

func TestLoadFileMixedFormats(t *testing.T) {
	src := `# header comment, ignored
SXIOPO            ; SMB infinite lives
00FF:42           # raw poke
6000:99:55        ; compare-gated poke

   0x6001:01      # 0x-prefixed address
$6002:$02:$AA     ; $-prefixed everything
`
	cheats, err := LoadFile(strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cheats) != 5 {
		t.Fatalf("got %d cheats, want 5: %+v", len(cheats), cheats)
	}

	// Game Genie line keeps its inline comment.
	if cheats[0].Comment != "SMB infinite lives" {
		t.Errorf("cheat[0] comment = %q, want %q", cheats[0].Comment, "SMB infinite lives")
	}

	// Raw poke AAAA:VV.
	if cheats[1].Address != 0x00FF || cheats[1].Value != 0x42 || cheats[1].HasCompare {
		t.Errorf("cheat[1] = %+v, want addr=00FF val=42 noCompare", cheats[1])
	}

	// Compare-gated AAAA:VV:CC.
	if cheats[2].Address != 0x6000 || cheats[2].Value != 0x99 ||
		!cheats[2].HasCompare || cheats[2].Compare != 0x55 {
		t.Errorf("cheat[2] = %+v, want 6000:99:55", cheats[2])
	}

	// 0x-prefixed address tolerated.
	if cheats[3].Address != 0x6001 || cheats[3].Value != 0x01 {
		t.Errorf("cheat[3] = %+v, want addr=6001 val=01", cheats[3])
	}

	// $-prefixed address/value/compare tolerated.
	if cheats[4].Address != 0x6002 || cheats[4].Value != 0x02 ||
		!cheats[4].HasCompare || cheats[4].Compare != 0xAA {
		t.Errorf("cheat[4] = %+v, want 6002:02:AA", cheats[4])
	}
}

func TestLoadFilePartialOnError(t *testing.T) {
	// A malformed line is reported via the error chain but must not discard
	// the good cheats around it.
	src := "00FF:42\nGARBAGE!!\n0001:01\n"
	cheats, err := LoadFile(strings.NewReader(src))
	if err == nil {
		t.Fatalf("expected error for malformed line, got nil")
	}
	if len(cheats) != 2 {
		t.Fatalf("got %d cheats, want 2 (the two valid lines)", len(cheats))
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error should name the bad line number, got %q", err.Error())
	}
}

func TestParseRawMalformed(t *testing.T) {
	cases := []string{
		"6000:11:22:33", // too many fields
		"ZZZZ:11",       // bad address hex
		"6000:ZZ",       // bad value hex
		"6000:11:ZZ",    // bad compare hex
	}
	for _, line := range cases {
		if _, err := LoadFile(strings.NewReader(line)); err == nil {
			t.Errorf("%q: expected error, got nil", line)
		}
	}
}

func TestSplitComment(t *testing.T) {
	cases := []struct {
		in, code, comment string
	}{
		{"SXIOPO ; lives", "SXIOPO ", "lives"},
		{"00FF:42 # poke", "00FF:42 ", "poke"},
		{"no comment here", "no comment here", ""},
		{"; only comment", "", "only comment"},
	}
	for _, tc := range cases {
		code, comment := splitComment(tc.in)
		if code != tc.code || comment != tc.comment {
			t.Errorf("splitComment(%q) = (%q,%q), want (%q,%q)",
				tc.in, code, comment, tc.code, tc.comment)
		}
	}
}

func TestManagerListAndString(t *testing.T) {
	m := NewManager()
	if m.Count() != 0 || !m.Enabled() {
		t.Fatalf("fresh manager: Count=%d Enabled=%v, want 0/true", m.Count(), m.Enabled())
	}

	m.Add(Cheat{Address: 0x1234, Value: 0x56})                                   // String -> "1234:56"
	m.Add(Cheat{Address: 0x1235, Value: 0x57, Compare: 0x58, HasCompare: true})  // -> "1235:57:58"
	m.Add(Cheat{Address: 0x1236, Value: 0x59, Source: "SXIOPO"})                 // -> source verbatim

	if m.Count() != 3 {
		t.Fatalf("Count = %d, want 3", m.Count())
	}
	list := m.List()
	if len(list) != 3 {
		t.Fatalf("List len = %d, want 3", len(list))
	}
	// Add sets Enabled on each stored cheat.
	for i, c := range list {
		if !c.Enabled {
			t.Errorf("list[%d] not enabled", i)
		}
	}

	want := []string{"1234:56", "1235:57:58", "SXIOPO"}
	for i, w := range want {
		if got := list[i].String(); got != w {
			t.Errorf("list[%d].String() = %q, want %q", i, got, w)
		}
	}
}
