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

	// cpuTicker caches the optional mapper.CPUTicker assertion so TickCPU
	// (called once per CPU instruction in nes.Step) doesn't pay an
	// interface-conversion cost on every dispatch.
	cpuTicker mapper.CPUTicker

	// audioSource caches the optional mapper.AudioSource assertion
	// (FME-7 expansion sound, etc.); the APU's mixer pulls per sample.
	audioSource mapper.AudioSource

	// hasExpansion is true when the mapper decodes the $4020-$5FFF
	// expansion-area window (MMC5). Memory uses this to decide
	// whether to route CPU R/W there to the mapper or leave open bus.
	hasExpansion bool

	// spriteCHRReader caches the optional mapper.SpriteCHRReader
	// assertion. When set, sprite pattern fetches route here instead
	// of through ReadCHR — used by MMC5's dual CHR set in 8×16 mode.
	spriteCHRReader mapper.SpriteCHRReader

	// spriteSizeHinter caches the optional mapper.SpriteSizeHinter so
	// PPU $2000 writes can tell the mapper whether 8×16 mode is on.
	spriteSizeHinter mapper.SpriteSizeHinter

	// scanlineNotifier caches the optional mapper.ScanlineNotifier
	// so the PPU can hand MMC5 explicit per-scanline ticks (A12
	// edges don't fire on games whose BG and sprites share a
	// pattern table).
	scanlineNotifier mapper.ScanlineNotifier

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

	mapperNumber := (cart.Header.Flags6 >> 4) | (cart.Header.Flags7 & 0xF0)

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
		// CHR RAM size — MMC3 games often expect 32KB, others use 8KB
		chrRAMSize := 8192
		if mapperNumber == 4 {
			chrRAMSize = 32768
		}
		cart.CHRRAM = make([]uint8, chrRAMSize)
	}

	// PRG RAM: 32KB for battery-backed carts (e.g. Final Fantasy II), 8KB
	// otherwise. We always allocate the 8KB window even when the iNES
	// header doesn't flag the cart as having RAM, because blargg's test
	// ROMs write their status protocol to $6000+ regardless of mapper or
	// battery flag, and many MMC3/MMC1 games use $6000-$7FFF as work RAM.
	if cart.Header.Flags6&0x02 != 0 {
		cart.PRGRAM = make([]uint8, 32768)
	} else {
		cart.PRGRAM = make([]uint8, 8192)
	}

	// Determine mirroring
	if cart.Header.Flags6&0x08 != 0 {
		cart.Mirroring = MirroringFourScreen
	} else if cart.Header.Flags6&0x01 != 0 {
		cart.Mirroring = MirroringVertical
	} else {
		cart.Mirroring = MirroringHorizontal
	}

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
	if t, ok := cart.Mapper.(mapper.CPUTicker); ok {
		cart.cpuTicker = t
	}
	if s, ok := cart.Mapper.(mapper.AudioSource); ok {
		cart.audioSource = s
	}
	if _, ok := cart.Mapper.(mapper.ExpansionDecoder); ok {
		cart.hasExpansion = true
	}
	if r, ok := cart.Mapper.(mapper.SpriteCHRReader); ok {
		cart.spriteCHRReader = r
	}
	if h, ok := cart.Mapper.(mapper.SpriteSizeHinter); ok {
		cart.spriteSizeHinter = h
	}
	if n, ok := cart.Mapper.(mapper.ScanlineNotifier); ok {
		cart.scanlineNotifier = n
	}

	return cart, nil
}

// HasExpansion reports whether the mapper decodes the $4020-$5FFF
// cartridge-expansion window.
func (c *Cartridge) HasExpansion() bool { return c.hasExpansion }

// ReadCHRSprite routes sprite pattern fetches through the mapper's
// sprite-specific CHR path when available (MMC5 8×16 mode), falling
// back to the unified ReadCHR for every other mapper.
func (c *Cartridge) ReadCHRSprite(addr uint16) uint8 {
	if c.spriteCHRReader != nil {
		return c.spriteCHRReader.ReadCHRSprite(addr)
	}
	return c.ReadCHR(addr)
}

// SetSpriteSize forwards PPU $2000 bit-5 writes to mappers that
// distinguish BG vs sprite CHR routing by sprite size.
func (c *Cartridge) SetSpriteSize(is8x16 bool) {
	if c.spriteSizeHinter != nil {
		c.spriteSizeHinter.SetSpriteSize(is8x16)
	}
}

// NotifyScanline tells the mapper that the PPU has just started a new
// rendering scanline. Used by MMC5 to drive its scanline-match IRQ.
func (c *Cartridge) NotifyScanline(scanline int, renderingEnabled bool) {
	if c.scanlineNotifier != nil {
		c.scanlineNotifier.NotifyScanline(scanline, renderingEnabled)
	}
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

// TickCPU forwards a CPU-cycle count to mappers whose internal timing
// runs on the CPU clock (FME-7's 16-bit IRQ counter). Mappers that
// don't implement mapper.CPUTicker get nothing — the assertion is
// resolved once in LoadFromReader.
func (c *Cartridge) TickCPU(cycles int) {
	if c.cpuTicker != nil {
		c.cpuTicker.TickCPU(cycles)
	}
}

// AudioSample returns the cartridge's expansion-sound mixer output
// (0 when the mapper has no audio chip). Used by the APU.
func (c *Cartridge) AudioSample() float32 {
	if c.audioSource != nil {
		return c.audioSource.AudioSample()
	}
	return 0
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
	if c.Mapper == nil {
		return
	}
	if n, ok := c.Mapper.(mapper.A12Notifier); ok {
		n.NotifyA12(chrAddr, renderingEnabled)
	}
}

// HasBattery reports whether the cartridge has battery-backed PRG RAM (iNES
// header flag 6 bit 1). Games with this flag (Zelda, Final Fantasy, etc.)
// expect SRAM contents to persist across power cycles.
func (c *Cartridge) HasBattery() bool {
	return c.Header.Flags6&0x02 != 0
}

// SaveState writes the cartridge's writable RAM regions (PRG RAM + CHR RAM)
// to w, then delegates mapper-internal register state to any mapper that
// implements mapper.Stateful. PRG/CHR ROM are immutable and re-loaded from
// disk; only volatile RAM and mapper registers need persisting.
func (c *Cartridge) SaveState(w io.Writer) error {
	if _, err := w.Write(c.PRGRAM); err != nil {
		return err
	}
	if _, err := w.Write(c.CHRRAM); err != nil {
		return err
	}
	if sm, ok := c.Mapper.(mapper.Stateful); ok {
		return sm.SaveState(w)
	}
	return nil
}

// LoadState restores PRG/CHR RAM contents in-place (so mappers that hold
// slice references to the same backing arrays see the update), then
// delegates to any mapper that implements mapper.Stateful.
func (c *Cartridge) LoadState(r io.Reader) error {
	if len(c.PRGRAM) > 0 {
		if _, err := io.ReadFull(r, c.PRGRAM); err != nil {
			return err
		}
	}
	if len(c.CHRRAM) > 0 {
		if _, err := io.ReadFull(r, c.CHRRAM); err != nil {
			return err
		}
	}
	if sm, ok := c.Mapper.(mapper.Stateful); ok {
		return sm.LoadState(r)
	}
	return nil
}

// SaveRAM writes the PRG RAM contents (battery-backed save data) to w.
// Returns nil if the cartridge has no PRG RAM allocated.
func (c *Cartridge) SaveRAM(w io.Writer) error {
	if len(c.PRGRAM) == 0 {
		return nil
	}
	_, err := w.Write(c.PRGRAM)
	return err
}

// LoadRAM reads battery-backed save data from r into PRG RAM. If r contains
// fewer bytes than PRG RAM, the tail is left untouched; extra bytes beyond
// PRG RAM are discarded. Returns nil if there's no PRG RAM to load into.
func (c *Cartridge) LoadRAM(r io.Reader) error {
	if len(c.PRGRAM) == 0 {
		return nil
	}
	_, err := io.ReadFull(r, c.PRGRAM)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		return nil
	}
	return err
}

// GetMirroring returns the current mirroring mode
func (c *Cartridge) GetMirroring() int {
	// Some mappers (like MMC1, MMC3) can change mirroring dynamically
	if m, ok := c.Mapper.(mapper.MirroringSource); ok {
		return int(m.GetMirroringMode())
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
