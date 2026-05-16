package mapper

// Mapper0 (NROM) - No mapping
type Mapper0 struct {
	cartridge *CartridgeData
}

// NewMapper0 creates a new Mapper0 instance
func NewMapper0(data *CartridgeData) *Mapper0 {
	return &Mapper0{cartridge: data}
}

// ReadPRG reads from PRG ROM
func (m *Mapper0) ReadPRG(addr uint16) uint8 {
	if addr >= 0x8000 {
		addr -= 0x8000
		if len(m.cartridge.PRGROM) == 16384 {
			// 16KB ROM, mirror at 0xC000
			addr = addr % 16384
		}
		if int(addr) < len(m.cartridge.PRGROM) {
			return m.cartridge.PRGROM[addr]
		}
		return 0
	}
	return readPRGRAM(m.cartridge, addr)
}

// WritePRG writes to PRG space
func (m *Mapper0) WritePRG(addr uint16, value uint8) {
	// ROM writes are ignored; only the $6000-$7FFF PRG RAM window is writable.
	writePRGRAM(m.cartridge, addr, value)
}

// ReadCHR reads from CHR ROM/RAM
func (m *Mapper0) ReadCHR(addr uint16) uint8 {
	return readCHRROMOrRAM(m.cartridge, addr)
}

// WriteCHR writes to CHR RAM
func (m *Mapper0) WriteCHR(addr uint16, value uint8) {
	// CHR ROM writes are ignored.
	writeCHRRAM(m.cartridge, addr, value)
}

// Step does nothing for Mapper0
func (m *Mapper0) Step() {
	// No special timing for NROM
}

// IsIRQPending returns false for Mapper0 (no IRQ support)
func (m *Mapper0) IsIRQPending() bool {
	return false
}

// ClearIRQ does nothing for Mapper0 (no IRQ support)
func (m *Mapper0) ClearIRQ() {
	// No IRQ to clear
}

// Mapper0 (NROM) has no internal state, so it intentionally does not
// implement mapper.Stateful. Save-state code skips the mapper section.