package mapper

// Shared address-decoding helpers for the simple memory regions every
// mapper handles the same way: the $6000-$7FFF PRG RAM window and the
// CHR ROM/RAM fallback paths. Bank-switching math (which is mapper-
// specific) is intentionally not shared here — each mapper still owns
// its own offset calculations.

// readPRGRAM returns data.PRGRAM[addr-0x6000] when addr is in $6000-$7FFF
// and PRG RAM is allocated; otherwise returns 0.
func readPRGRAM(data *CartridgeData, addr uint16) uint8 {
	if addr < 0x6000 || addr >= 0x8000 || len(data.PRGRAM) == 0 {
		return 0
	}
	i := int(addr - 0x6000)
	if i >= len(data.PRGRAM) {
		return 0
	}
	return data.PRGRAM[i]
}

// writePRGRAM writes value to PRG RAM at addr-$6000 when in range and
// PRG RAM is allocated; otherwise a no-op.
func writePRGRAM(data *CartridgeData, addr uint16, value uint8) {
	if addr < 0x6000 || addr >= 0x8000 || len(data.PRGRAM) == 0 {
		return
	}
	i := int(addr - 0x6000)
	if i >= len(data.PRGRAM) {
		return
	}
	data.PRGRAM[i] = value
}

// readCHRROMOrRAM returns CHRROM[addr] when CHR ROM is present, else
// CHRRAM[addr] when CHR RAM is present, else 0. Bounds-checked.
func readCHRROMOrRAM(data *CartridgeData, addr uint16) uint8 {
	if len(data.CHRROM) > 0 {
		if int(addr) < len(data.CHRROM) {
			return data.CHRROM[addr]
		}
		return 0
	}
	if len(data.CHRRAM) > 0 {
		if int(addr) < len(data.CHRRAM) {
			return data.CHRRAM[addr]
		}
	}
	return 0
}

// writeCHRRAM writes to CHR RAM when present, no-op otherwise.
func writeCHRRAM(data *CartridgeData, addr uint16, value uint8) {
	if len(data.CHRRAM) == 0 {
		return
	}
	if int(addr) < len(data.CHRRAM) {
		data.CHRRAM[addr] = value
	}
}
