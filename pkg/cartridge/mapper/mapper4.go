package mapper

import (
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

// ReadPRG reads from PRG ROM/RAM address space
func (m *Mapper4) ReadPRG(addr uint16) uint8 {
	switch {
	case addr >= 0x6000 && addr <= 0x7FFF:
		// PRG RAM area - check both read enable and write protect
		if len(m.data.PRGRAM) > 0 && (m.prgRAMProtect&0x80) != 0 {
			return m.data.PRGRAM[addr-0x6000]
		}
		return 0

	case addr >= 0x8000 && addr <= 0xFFFF:
		// Correct MMC3 PRG ROM banking according to NESdev wiki
		var bank uint8
		prgMode := (m.bankSelect >> 6) & 1

		switch {
		case addr >= 0x8000 && addr <= 0x9FFF:
			// $8000-$9FFF: R6 in mode 0, second-to-last in mode 1
			if prgMode == 0 {
				bank = m.bankRegisters[6]
			} else {
				bank = m.prgBankCount - 2 // second-to-last bank
			}

		case addr >= 0xA000 && addr <= 0xBFFF:
			// $A000-$BFFF: Always R7
			bank = m.bankRegisters[7]

		case addr >= 0xC000 && addr <= 0xDFFF:
			// $C000-$DFFF: second-to-last in mode 0, R6 in mode 1
			if prgMode == 0 {
				bank = m.prgBankCount - 2 // second-to-last bank
			} else {
				bank = m.bankRegisters[6]
			}

		case addr >= 0xE000 && addr <= 0xFFFF:
			// $E000-$FFFF: Always last bank (this is crucial for reset vector)
			bank = m.prgBankCount - 1
		}

		// Ensure bank is in valid range
		if bank >= m.prgBankCount {
			bank = m.prgBankCount - 1
		}

		// Calculate offset within PRG ROM (8KB banks)
		offset := uint32(bank)*0x2000 + uint32(addr&0x1FFF)
		if offset < uint32(len(m.data.PRGROM)) {
			return m.data.PRGROM[offset]
		}
	}

	return 0
}

// WritePRG writes to PRG ROM/RAM address space or mapper registers
func (m *Mapper4) WritePRG(addr uint16, value uint8) {
	switch {
	case addr >= 0x6000 && addr <= 0x7FFF:
		// PRG RAM area - check write protection
		if len(m.data.PRGRAM) > 0 && (m.prgRAMProtect&0x80) != 0 && (m.prgRAMProtect&0x40) == 0 {
			m.data.PRGRAM[addr-0x6000] = value
		}

	case addr >= 0x8000:
		// Enable essential MMC3 registers according to NESdev spec
		switch addr & 0xE001 {
		case 0x8000: // Bank select ($8000-$9FFE, even)
			logger.LogMapper("MMC3 bank select set: %d", value)
			m.bankSelect = value

		case 0x8001: // Bank data ($8001-$9FFF, odd)
			regIndex := m.bankSelect & 0x07
			if regIndex < 8 {
				// Bounds check to prevent invalid bank access
				if regIndex >= 6 {
					// PRG bank registers (R6, R7) - clamp to valid range
					m.bankRegisters[regIndex] = value % m.prgBankCount
				} else {
					// CHR bank registers (R0-R5) - clamp to valid range
					if m.chrBankCount > 0 {
						m.bankRegisters[regIndex] = value % m.chrBankCount
					} else {
						m.bankRegisters[regIndex] = value
					}
				}
			}

		case 0xA000: // Mirroring ($A000-$BFFE, even)
			m.mirroringMode = value & 1

		case 0xA001: // PRG RAM protect ($A001-$BFFF, odd)
			m.prgRAMProtect = value

		case 0xC000: // IRQ latch ($C000-$DFFE, even)
			m.irqReloadValue = value
			logger.LogMapper("MMC3 IRQ latch set: %d", value)

		case 0xC001: // IRQ reload ($C001-$DFFF, odd)
			m.irqReloadFlag = true
			m.irqCounter = 0 // Clear counter
			logger.LogMapper("MMC3 IRQ reload triggered")

		case 0xE000: // IRQ disable ($E000-$FFFE, even)
			m.irqEnabled = false
			m.irqPending = false
			logger.LogMapper("MMC3 IRQ disabled")

		case 0xE001: // IRQ enable ($E001-$FFFF, odd)
			m.irqEnabled = true
			logger.LogMapper("MMC3 IRQ enabled")
		}
	}
}

// ReadCHR reads from CHR ROM/RAM address space
func (m *Mapper4) ReadCHR(addr uint16) uint8 {
	if addr >= 0x2000 {
		return 0
	}

	// Use common CHR banking calculation
	bank := m.calculateCHRBank(addr)

	// Handle CHR ROM with banking
	if len(m.data.CHRROM) > 0 {
		// Ensure bank is in valid range
		if m.chrBankCount > 0 {
			bank %= m.chrBankCount
		}
		offset := uint32(bank)*0x400 + uint32(addr&0x3FF)
		if offset < uint32(len(m.data.CHRROM)) {
			return m.data.CHRROM[offset]
		}
	}

	// Handle CHR RAM with banking (32KB support)
	if len(m.data.CHRRAM) > 0 {
		if m.chrBankCount > 0 {
			bank %= m.chrBankCount
		}
		offset := uint32(bank)*0x400 + uint32(addr&0x3FF)
		if offset < uint32(len(m.data.CHRRAM)) {
			return m.data.CHRRAM[offset]
		} else {
			logger.LogMapper("CHR Write ERROR: addr=$%04X, bank=%d, offset=$%06X >= size=%d",
				addr, bank, offset, len(m.data.CHRRAM))
		}
	}

	return 0
}

// calculateCHRBank calculates the CHR bank number for a given address
func (m *Mapper4) calculateCHRBank(addr uint16) uint8 {
	var bank uint8
	chrMode := (m.bankSelect >> 7) & 1

	if chrMode == 0 {
		// Mode 0: $0000-$0FFF = R0,R1 (2KB each), $1000-$1FFF = R2,R3,R4,R5 (1KB each)
		if addr < 0x1000 {
			if addr < 0x800 {
				// $0000-$07FF: R0 (2KB bank) - even bank only, lowest bit ignored
				bank = (m.bankRegisters[0] &^ 1) + uint8(addr/0x400)
			} else {
				// $0800-$0FFF: R1 (2KB bank) - even bank only, lowest bit ignored
				bank = (m.bankRegisters[1] &^ 1) + uint8((addr-0x800)/0x400)
			}
		} else {
			// $1000-$1FFF: R2,R3,R4,R5 (1KB each)
			regIndex := 2 + (addr-0x1000)/0x400
			bank = m.bankRegisters[regIndex]
		}
	} else {
		// Mode 1: $0000-$0FFF = R2,R3,R4,R5 (1KB each), $1000-$1FFF = R0,R1 (2KB each)
		if addr < 0x1000 {
			// $0000-$0FFF: R2,R3,R4,R5 (1KB each)
			regIndex := 2 + addr/0x400
			bank = m.bankRegisters[regIndex]
		} else {
			if addr < 0x1800 {
				// $1000-$17FF: R0 (2KB bank) - even bank only, lowest bit ignored
				bank = (m.bankRegisters[0] &^ 1) + uint8((addr-0x1000)/0x400)
			} else {
				// $1800-$1FFF: R1 (2KB bank) - even bank only, lowest bit ignored
				bank = (m.bankRegisters[1] &^ 1) + uint8((addr-0x1800)/0x400)
			}
		}
	}

	return bank
}

// WriteCHR writes to CHR ROM/RAM address space
func (m *Mapper4) WriteCHR(addr uint16, value uint8) {
	if addr >= 0x2000 {
		return
	}

	// Only CHR RAM is writable - with banking support
	if len(m.data.CHRRAM) > 0 {
		// Use common CHR banking calculation
		bank := m.calculateCHRBank(addr)

		// Ensure bank is in valid range for CHR RAM
		if m.chrBankCount > 0 {
			bank %= m.chrBankCount
		}
		offset := uint32(bank)*0x400 + uint32(addr&0x3FF)
		if offset < uint32(len(m.data.CHRRAM)) {
			m.data.CHRRAM[offset] = value
		}
	}
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

// NotifyA12 handles A12 rising edge for MMC3 IRQ timing
func (m *Mapper4) NotifyA12(chrAddr uint16, renderingEnabled bool) {
	// Only process A12 transitions when rendering is enabled
	if !renderingEnabled {
		return
	}

	a12High := (chrAddr & 0x1000) != 0

	// Research-level A12 filtering implementation
	// Based on extensive analysis of MMC3 chip behavior

	// Update M2 cycle counter
	m.m2CycleCount++

	// Store A12 state in circular history buffer
	m.a12History[m.a12HistoryPos] = a12High
	m.a12HistoryPos = (m.a12HistoryPos + 1) % 8

	// Implement precise filtering: "after the line has remained low for three falling edges of M2"
	// This means A12 must be low for at least 3 M2 cycles (6 PPU cycles)
	if !a12High {
		// A12 is low - count consecutive low states
		m.a12LowCounter++
		// Check if we have sufficient consecutive low states in history
		consecutiveLowCount := 0
		for i := 0; i < 8; i++ {
			if !m.a12History[i] {
				consecutiveLowCount++
			} else {
				break
			}
		}

		// Filter pass requires at least 6 consecutive low states (3 M2 falling edges)
		if consecutiveLowCount >= 6 {
			m.a12FilterPass = true
		}
	} else {
		// A12 is high - check for valid rising edge
		if m.a12Low && m.a12FilterPass {
			// Additional validation: ensure this is a genuine rising edge
			// Check that previous states were consistently low
			validTransition := true
			for i := 1; i < 4; i++ { // Check last 3 states
				prevPos := (m.a12HistoryPos - i + 8) % 8
				if m.a12History[prevPos] {
					validTransition = false
					break
				}
			}

			if validTransition {
				// Valid rising edge detected - clock IRQ counter
				m.Step()
			}
		}
		// Reset filter state when A12 goes high
		m.a12LowCounter = 0
		m.a12FilterPass = false
	}

	m.a12Low = !a12High
}

// IsIRQPending returns true if an IRQ is pending
func (m *Mapper4) IsIRQPending() bool {
	return m.irqPending
}

// ClearIRQ clears the pending IRQ
func (m *Mapper4) ClearIRQ() {
	m.irqPending = false
}

// GetMirroringMode returns the current mirroring mode
func (m *Mapper4) GetMirroringMode() uint8 {
	return m.mirroringMode
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
