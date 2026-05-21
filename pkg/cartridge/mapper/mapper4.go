package mapper

// Mapper4 (MMC3) is split across three files:
//
//   - mapper4.go        : Mapper4 struct, constructor, Step(), mirroring,
//                         debug helpers, SaveState/LoadState.
//   - mapper4_banks.go  : PRG/CHR bank mapping (ReadPRG/WritePRG/ReadCHR/
//                         WriteCHR/recalcCHRBanks). WritePRG hosts the
//                         outer $E001 switch and delegates IRQ-register
//                         arms to helpers in mapper4_irq.go.
//   - mapper4_irq.go    : A12 IRQ notification stub, IRQ register-write
//                         helpers, and IRQ status API (IsIRQPending /
//                         ClearIRQ).

import (
	"encoding/binary"
	"io"

	"github.com/yoshiomiyamaegones/pkg/logger"
)

// Mapper4 implements the MMC3 (Multi-Memory Controller 3) mapper
// This is one of the most complex and widely used mappers in NES games
type Mapper4 struct {
	data *CartridgeData

	// Bank registers R0-R7
	bankRegisters [8]uint8

	// Bank select register controls which register to update and banking modes
	bankSelect uint8

	// Mirroring mode (0 = vertical, 1 = horizontal)
	mirroringMode uint8

	// PRG RAM control
	prgRAMProtect uint8

	// IRQ timer
	irqReloadValue uint8
	irqCounter     uint8
	irqEnabled     bool
	irqPending     bool
	irqReloadFlag  bool // Set when $C001 is written

	// lastA12High tracks the A12 line state across CPU-driven PPU register
	// accesses ($2006 second-write, $2007 R/W increments). NotifyA12 reads
	// this to detect 0→1 transitions; Step() (per-scanline path) resets it
	// to false so the next CPU-driven A12 high write is treated as a fresh
	// rising edge rather than being suppressed by a stale "still high".
	lastA12High bool

	// Bank counts
	prgBankCount uint8
	chrBankCount uint8

	// chrWindowOffset holds the byte offset into the CHR backing array for
	// each of the eight 1KB CHR windows ($0000-$1FFF). The MMC3 CHR mapping
	// only changes on $8000 (bank select / mode) and $8001 (bank data)
	// writes, so resolving it per fetch is wasted work — ReadCHR/WriteCHR
	// index this table instead. Rebuilt by recalcCHRBanks at init, on those
	// writes, and after LoadState.
	chrWindowOffset [8]uint32
}

// NewMapper4 creates a new MMC3 mapper instance
func NewMapper4(data *CartridgeData) *Mapper4 {
	m := &Mapper4{
		data:          data,
		prgBankCount:  uint8(len(data.PRGROM) / 8192), // 8KB banks
		prgRAMProtect: 0x80,                           // PRG RAM enabled by default
	}

	// Set CHR bank count based on available CHR ROM or RAM
	if len(data.CHRROM) > 0 {
		m.chrBankCount = uint8(len(data.CHRROM) / 1024) // 1KB banks
		logger.LogMapper("MMC3 initialized with CHR ROM: %d bytes, %d banks", len(data.CHRROM), m.chrBankCount)
	} else if len(data.CHRRAM) > 0 {
		m.chrBankCount = uint8(len(data.CHRRAM) / 1024) // 1KB banks
		logger.LogMapper("MMC3 initialized with CHR RAM: %d bytes, %d banks", len(data.CHRRAM), m.chrBankCount)
	} else {
		m.chrBankCount = 8 // Default to 8KB (8x1KB banks)
		logger.LogMapper("MMC3 initialized with default CHR: %d banks", m.chrBankCount)
	}

	// Initialize bank registers to safe default values
	// R6 and R7 are set to the last two PRG banks
	if m.prgBankCount >= 2 {
		m.bankRegisters[6] = m.prgBankCount - 2
		m.bankRegisters[7] = m.prgBankCount - 1
	}

	// Initialize CHR bank registers to safe values
	for i := 0; i < 6; i++ {
		if m.chrBankCount > 0 {
			m.bankRegisters[i] = uint8(i % int(m.chrBankCount)) // Safe initial CHR banks within bounds
		} else {
			m.bankRegisters[i] = uint8(i) // Default values
		}
		logger.LogMapper("MMC3 CHR bank R%d initialized to %d", i, m.bankRegisters[i])
	}

	logger.LogInfo("CHR RAM initialized: size=%d bytes", len(data.CHRRAM))

	m.recalcCHRBanks()

	return m
}

// Step is invoked by the PPU at the per-scanline A12-rising-edge tick.
// Resets lastA12High so a CPU $2006 = $1xxx write between scanlines is
// recognised as a fresh rising edge by NotifyA12 instead of being
// suppressed as "still high".
func (m *Mapper4) Step() {
	m.clockIRQ()
	m.lastA12High = false
}

// GetMirroringMode returns the current mirroring mode in the PPU's encoding
// (0 = horizontal mirroring = $2000 mirrors $2400; 1 = vertical mirroring =
// $2000 mirrors $2800). The MMC3 wiki labels its $A000 bit 0 in *arrangement*
// terminology, which is the inverse of mirroring terminology:
//
//   MMC3 wiki    | Arrangement | Mirroring (PPU code) | Physical effect
//   bit 0 = 0    | horizontal  | vertical             | $2000 = $2800 (horiz scroll)
//   bit 0 = 1    | vertical    | horizontal           | $2000 = $2400 (vert scroll)
//
// So we invert the stored MMC3 bit to get the PPU's mirroring encoding.
func (m *Mapper4) GetMirroringMode() uint8 {
	if m.mirroringMode == 0 {
		return 1 // MMC3 horizontal arrangement -> PPU vertical mirroring
	}
	return 0 // MMC3 vertical arrangement -> PPU horizontal mirroring
}

// GetBankSelect returns the current bank select register for debugging
func (m *Mapper4) GetBankSelect() uint8 {
	return m.bankSelect
}

// GetBankRegisters returns the current bank registers for debugging
func (m *Mapper4) GetBankRegisters() [8]uint8 {
	return m.bankRegisters
}

// GetIRQState returns current IRQ state for debugging
func (m *Mapper4) GetIRQState() (uint8, uint8, bool, bool) {
	return m.irqCounter, m.irqReloadValue, m.irqEnabled, m.irqPending
}

// GetCurrentPRGBanks returns the current PRG bank configuration for debugging
func (m *Mapper4) GetCurrentPRGBanks() [4]uint8 {
	var banks [4]uint8
	prgMode := (m.bankSelect >> 6) & 1

	if prgMode == 0 {
		banks[0] = m.bankRegisters[6]
		banks[1] = m.bankRegisters[7]
		banks[2] = m.prgBankCount - 2
		banks[3] = m.prgBankCount - 1
	} else {
		banks[0] = m.prgBankCount - 2
		banks[1] = m.bankRegisters[7]
		banks[2] = m.bankRegisters[6]
		banks[3] = m.prgBankCount - 1
	}

	return banks
}

// GetDebugInfo returns detailed debug information for Mapper 4
func (m *Mapper4) GetDebugInfo() map[string]interface{} {
	return map[string]interface{}{
		"bankSelect":     m.bankSelect,
		"bankRegisters":  m.bankRegisters,
		"prgMode":        (m.bankSelect >> 6) & 1,
		"chrMode":        (m.bankSelect >> 7) & 1,
		"mirroringMode":  m.mirroringMode,
		"prgRAMProtect":  m.prgRAMProtect,
		"irqReloadValue": m.irqReloadValue,
		"irqCounter":     m.irqCounter,
		"irqEnabled":     m.irqEnabled,
		"irqPending":     m.irqPending,
		"prgBankCount":   m.prgBankCount,
		"chrBankCount":   m.chrBankCount,
	}
}

// DumpCHRRAM dumps first 32 bytes of each CHR bank for debugging
func (m *Mapper4) DumpCHRRAM() {
	if len(m.data.CHRRAM) > 0 {
		logger.LogMapper("CHR RAM Dump (first 8 banks):")
		for bank := 0; bank < 8 && bank < int(m.chrBankCount); bank++ {
			offset := bank * 0x400
			if offset+16 < len(m.data.CHRRAM) {
				data := m.data.CHRRAM[offset : offset+16]
				logger.LogMapper("Bank %d: %02X %02X %02X %02X %02X %02X %02X %02X %02X %02X %02X %02X %02X %02X %02X %02X",
					bank, data[0], data[1], data[2], data[3], data[4], data[5], data[6], data[7],
					data[8], data[9], data[10], data[11], data[12], data[13], data[14], data[15])
			}
		}
	}
}

// TriggerDump triggers a CHR RAM dump when called
func (m *Mapper4) TriggerDump() {
	m.DumpCHRRAM()
}

// mapper4State persists all runtime state. lastA12High is intentionally
// excluded — it's a CPU-side transient (re-derived from the next $2006
// write) and snapshotting it across save states would freeze the next
// edge detection at the wrong reference.
type mapper4State struct {
	BankRegisters  [8]uint8
	BankSelect     uint8
	MirroringMode  uint8
	PrgRAMProtect  uint8
	IrqReloadValue uint8
	IrqCounter     uint8
	IrqEnabled     bool
	IrqPending     bool
	IrqReloadFlag  bool
}

// SaveState writes MMC3 register/IRQ state to w.
func (m *Mapper4) SaveState(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, mapper4State{
		BankRegisters:  m.bankRegisters,
		BankSelect:     m.bankSelect,
		MirroringMode:  m.mirroringMode,
		PrgRAMProtect:  m.prgRAMProtect,
		IrqReloadValue: m.irqReloadValue,
		IrqCounter:     m.irqCounter,
		IrqEnabled:     m.irqEnabled,
		IrqPending:     m.irqPending,
		IrqReloadFlag:  m.irqReloadFlag,
	})
}

// LoadState restores MMC3 state written by SaveState.
func (m *Mapper4) LoadState(r io.Reader) error {
	var s mapper4State
	if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
		return err
	}
	m.bankRegisters = s.BankRegisters
	m.bankSelect = s.BankSelect
	m.mirroringMode = s.MirroringMode
	m.prgRAMProtect = s.PrgRAMProtect
	m.irqReloadValue = s.IrqReloadValue
	m.irqCounter = s.IrqCounter
	m.irqEnabled = s.IrqEnabled
	m.irqPending = s.IrqPending
	m.irqReloadFlag = s.IrqReloadFlag
	m.recalcCHRBanks() // rebuild window table from restored bank registers
	return nil
}
