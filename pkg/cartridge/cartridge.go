package cartridge

import (
	"fmt"
	"io"

	"github.com/yoshiomiyamaegones/pkg/cartridge/mapper"
)

// Cartridge represents a NES cartridge
type Cartridge struct {
	// ROM data
	PRGROM []uint8 // Program ROM
	CHRROM []uint8 // Character ROM

	// RAM data
	PRGRAM []uint8 // Program RAM (SRAM)
	CHRRAM []uint8 // Character RAM

	// Header information
	Header iNESHeader

	// Mapper
	Mapper mapper.Mapper

	// Mirroring
	Mirroring MirroringMode
}

// iNESHeader represents the iNES file header
type iNESHeader struct {
	Magic      [4]uint8 // "NES\x1A"
	PRGROMSize uint8    // Size of PRG ROM in 16KB units
	CHRROMSize uint8    // Size of CHR ROM in 8KB units
	Flags6     uint8    // Mapper, mirroring, battery, trainer
	Flags7     uint8    // Mapper, VS/Playchoice, NES 2.0
	Flags8     uint8    // PRG-RAM size (rarely used)
	Flags9     uint8    // TV system (rarely used)
	Flags10    uint8    // TV system, PRG-RAM presence (unofficial)
	Padding    [5]uint8 // Unused padding (should be zero)
}

// MirroringMode represents the mirroring mode
type MirroringMode int

const (
	MirroringHorizontal MirroringMode = iota
	MirroringVertical
	MirroringFourScreen
	MirroringSingleScreenA
	MirroringSingleScreenB
)

// LoadFromReader loads a cartridge from an iNES file
func LoadFromReader(reader io.Reader) (*Cartridge, error) {
	cart := &Cartridge{}

	// Read header
	err := cart.readHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Validate header
	if string(cart.Header.Magic[:]) != "NES\x1A" {
		return nil, fmt.Errorf("invalid iNES magic number")
	}

	// Skip trainer if present
	if cart.Header.Flags6&0x04 != 0 {
		trainer := make([]uint8, 512)
		_, err := io.ReadFull(reader, trainer)
		if err != nil {
			return nil, fmt.Errorf("failed to read trainer: %w", err)
		}
	}

	// Read PRG ROM
	prgSize := int(cart.Header.PRGROMSize) * 16384
	cart.PRGROM = make([]uint8, prgSize)
	_, err = io.ReadFull(reader, cart.PRGROM)
	if err != nil {
		return nil, fmt.Errorf("failed to read PRG ROM: %w", err)
	}

	// Read CHR ROM
	chrSize := int(cart.Header.CHRROMSize) * 8192
	if chrSize > 0 {
		cart.CHRROM = make([]uint8, chrSize)
		_, err = io.ReadFull(reader, cart.CHRROM)
		if err != nil {
			return nil, fmt.Errorf("failed to read CHR ROM: %w", err)
		}
	} else {
		// CHR RAM - determine size based on mapper
		mapperNumber := (cart.Header.Flags6 >> 4) | (cart.Header.Flags7 & 0xF0)
		chrRAMSize := 8192 // Default 8KB

		// Mapper 4 (MMC3) games often use 32KB CHR RAM
		if mapperNumber == 4 {
			chrRAMSize = 32768 // 32KB for MMC3 games
		}

		cart.CHRRAM = make([]uint8, chrRAMSize)

		// Initialize CHR RAM to 0x00 (normal expected state)
		for i := range cart.CHRRAM {
			cart.CHRRAM[i] = 0x00
		}
	}

	// Initialize PRG RAM if battery backed
	if cart.Header.Flags6&0x02 != 0 {
		// Final Fantasy II requires 32KB PRG RAM, not 8KB
		cart.PRGRAM = make([]uint8, 32768)
	}

	// Determine mirroring
	if cart.Header.Flags6&0x08 != 0 {
		cart.Mirroring = MirroringFourScreen
	} else if cart.Header.Flags6&0x01 != 0 {
		cart.Mirroring = MirroringVertical
	} else {
		cart.Mirroring = MirroringHorizontal
	}

	// Create mapper
	mapperNumber := (cart.Header.Flags6 >> 4) | (cart.Header.Flags7 & 0xF0)

	// Create mapper data
	mapperData := &mapper.CartridgeData{
		PRGROM: cart.PRGROM,
		CHRROM: cart.CHRROM,
		PRGRAM: cart.PRGRAM,
		CHRRAM: cart.CHRRAM,
	}

	cart.Mapper, err = mapper.NewMapper(mapperNumber, mapperData)
	if err != nil {
		return nil, fmt.Errorf("failed to create mapper: %w", err)
	}

	return cart, nil
}

// readHeader reads the iNES header
func (c *Cartridge) readHeader(reader io.Reader) error {
	headerBytes := make([]uint8, 16)
	_, err := io.ReadFull(reader, headerBytes)
	if err != nil {
		return err
	}

	copy(c.Header.Magic[:], headerBytes[0:4])
	c.Header.PRGROMSize = headerBytes[4]
	c.Header.CHRROMSize = headerBytes[5]
	c.Header.Flags6 = headerBytes[6]
	c.Header.Flags7 = headerBytes[7]
	c.Header.Flags8 = headerBytes[8]
	c.Header.Flags9 = headerBytes[9]
	c.Header.Flags10 = headerBytes[10]
	copy(c.Header.Padding[:], headerBytes[11:16])

	return nil
}

// ReadPRG reads from PRG space
func (c *Cartridge) ReadPRG(addr uint16) uint8 {
	if c.Mapper != nil {
		return c.Mapper.ReadPRG(addr)
	}
	return 0
}

// WritePRG writes to PRG space
func (c *Cartridge) WritePRG(addr uint16, value uint8) {
	if c.Mapper != nil {
		c.Mapper.WritePRG(addr, value)
	}
}

// ReadCHR reads from CHR space
func (c *Cartridge) ReadCHR(addr uint16) uint8 {
	if c.Mapper != nil {
		return c.Mapper.ReadCHR(addr)
	}
	return 0
}

// WriteCHR writes to CHR space
func (c *Cartridge) WriteCHR(addr uint16, value uint8) {
	if c.Mapper != nil {
		c.Mapper.WriteCHR(addr, value)
	}
}

// Step steps the mapper (for mappers with timing)
func (c *Cartridge) Step() {
	if c.Mapper != nil {
		c.Mapper.Step()
	}
}

// IsIRQPending returns whether mapper IRQ is pending
func (c *Cartridge) IsIRQPending() bool {
	if c.Mapper != nil {
		return c.Mapper.IsIRQPending()
	}
	return false
}

// ClearIRQ clears mapper IRQ
func (c *Cartridge) ClearIRQ() {
	if c.Mapper != nil {
		c.Mapper.ClearIRQ()
	}
}

// NotifyA12 notifies the mapper of A12 line state for MMC3 IRQ timing
func (c *Cartridge) NotifyA12(chrAddr uint16, renderingEnabled bool) {
	if c.Mapper != nil {
		// Check if mapper supports A12 notification (MMC3/Mapper4)
		if mapper4, ok := c.Mapper.(*mapper.Mapper4); ok {
			mapper4.NotifyA12(chrAddr, renderingEnabled)
		}
	}
}

// GetMirroring returns the current mirroring mode
func (c *Cartridge) GetMirroring() int {
	// Some mappers (like MMC1, MMC3) can change mirroring dynamically
	if mapper, ok := c.Mapper.(interface{ GetMirroringMode() uint8 }); ok {
		return int(mapper.GetMirroringMode())
	}

	// Fall back to cartridge header mirroring
	switch c.Mirroring {
	case MirroringHorizontal:
		return 0
	case MirroringVertical:
		return 1
	default:
		return 0 // Default to horizontal
	}
}
