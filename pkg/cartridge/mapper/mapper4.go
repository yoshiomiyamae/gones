package mapper

// Mapper4 (MMC3) is split across three files:
//
//   - mapper4.go        : Mapper4 struct, constructor, Step(), mirroring,
//                         debug helpers, SaveState/LoadState.
//   - mapper4_banks.go  : PRG/CHR bank mapping (ReadPRG/WritePRG/ReadCHR/
//                         WriteCHR/calculateCHRBank). WritePRG hosts the
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

	// A12 filtering for proper IRQ timing according to NESdev spec
	a12Low        bool
	a12LowCounter int     // Count M2 cycles while A12 is low
	a12FilterPass bool    // Flag indicating A12 has been low for sufficient time
	a12History    [8]bool // History of A12 states for precise filtering
	a12HistoryPos int     // Position in A12 history buffer
	m2CycleCount  int     // M2 clock cycle counter for precise timing

	// Bank counts
	prgBankCount uint8
	chrBankCount uint8

	// MMC3 chip variant (true = Sharp MMC3, false = NEC MMC3)
	isSharpMMC3 bool
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

	// Initialize IRQ counter with MMC3 defaults
	m.irqCounter = 0        // Start with 0 as per MMC3 spec
	m.irqReloadValue = 0    // Default reload value
	m.irqEnabled = false    // IRQ disabled by default
	m.irqPending = false    // No pending IRQ
	m.irqReloadFlag = false // No reload pending
	m.a12Low = true         // Initialize A12 state
	m.a12LowCounter = 0     // Initialize A12 low counter
	m.a12FilterPass = false // Initialize A12 filter state
	m.a12HistoryPos = 0     // Initialize history position
	m.m2CycleCount = 0      // Initialize M2 cycle counter
	m.isSharpMMC3 = true    // Default to Sharp MMC3 behavior (more common)

	logger.LogInfo("CHR RAM initialized: size=%d bytes", len(data.CHRRAM))

	return m
}

// Step advances the IRQ timer - called on A12 rising edge
func (m *Mapper4) Step() {
	// MMC3 IRQ counter decrements on A12 rising edge when rendering is enabled
	if m.irqReloadFlag {
		m.irqCounter = m.irqReloadValue
		m.irqReloadFlag = false
	} else if m.irqCounter == 0 {
		m.irqCounter = m.irqReloadValue
	} else {
		m.irqCounter--
	}

	// Trigger IRQ based on chip variant behavior
	var shouldTriggerIRQ bool
	if m.isSharpMMC3 {
		// Sharp MMC3: Generates an IRQ on each scanline when counter reaches 0
		shouldTriggerIRQ = (m.irqCounter == 0 && m.irqEnabled)
	} else {
		// NEC MMC3: Generates only a single IRQ when counter reaches 0
		// (More complex behavior - simplified for now)
		shouldTriggerIRQ = (m.irqCounter == 0 && m.irqEnabled && m.irqReloadValue > 0)
	}

	if shouldTriggerIRQ {
		m.irqPending = true
		logger.LogMapper("MMC3 IRQ triggered (reload=%d, variant=%s)",
			m.irqReloadValue, func() string {
				if m.isSharpMMC3 {
					return "Sharp"
				}
				return "NEC"
			}())
	}
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

// mapper4State persists all runtime state. a12History/a12HistoryPos are no
// longer used by the IRQ path (PPU drives Step() at cycle 260) but are kept
// in the snapshot for stability with future revisions.
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
	return nil
}
