package mapper

import (
	"fmt"
	"io"
)

// Mapper interface for different mappers
type Mapper interface {
	ReadPRG(addr uint16) uint8
	WritePRG(addr uint16, value uint8)
	ReadCHR(addr uint16) uint8
	WriteCHR(addr uint16, value uint8)
	Step()
	IsIRQPending() bool
	ClearIRQ()
}

// Stateful is the optional save-state hook for mappers with persistent
// runtime state (bank registers, IRQ counters, etc.). Mappers without
// internal state (e.g. NROM) don't need to implement it — callers should
// type-assert before invoking.
type Stateful interface {
	SaveState(w io.Writer) error
	LoadState(r io.Reader) error
}

// A12Notifier is the optional interface for mappers that need PPU A12-line
// edge notifications (used by MMC3 for its scanline IRQ counter). Mappers
// without IRQ scanline timing don't implement this and the cartridge layer
// simply skips the call.
type A12Notifier interface {
	NotifyA12(chrAddr uint16, renderingEnabled bool)
}

// CPUTicker is the optional interface for mappers whose internal timing
// runs on the CPU clock (e.g. Sunsoft FME-7's 16-bit IRQ counter, which
// decrements every CPU cycle). nes.Step calls TickCPU once per
// completed CPU instruction with that instruction's cycle count;
// mappers without CPU-rate timing don't implement it.
type CPUTicker interface {
	TickCPU(cycles int)
}

// AudioSource is the optional interface for mappers with expansion
// sound (FME-7's three square channels, VRC6/VRC7 etc. once those
// land). The APU's mixer pulls AudioSample() per output sample and
// mixes the value into the 2A03 channel sum. Range is the same 0..1
// the APU uses internally.
type AudioSource interface {
	AudioSample() float32
}

// MirroringSource is the optional interface for mappers that override the
// iNES-header mirroring mode dynamically (MMC1, MMC3 — anything with a
// mirroring register). Falls back to the header value when the mapper
// doesn't implement it.
type MirroringSource interface {
	GetMirroringMode() uint8
}

// CartridgeData contains cartridge data for mappers
type CartridgeData struct {
	PRGROM []uint8
	CHRROM []uint8
	PRGRAM []uint8
	CHRRAM []uint8
}

// NewMapper creates a new mapper instance
func NewMapper(mapperNumber uint8, data *CartridgeData) (Mapper, error) {
	switch mapperNumber {
	case 0:
		return NewMapper0(data), nil
	case 1:
		return NewMapper1(data), nil
	case 2:
		return NewMapper2(data), nil
	case 3:
		return NewMapper3(data), nil
	case 4:
		return NewMapper4(data), nil
	case 10:
		return NewMapper10(data), nil
	case 69:
		return NewMapper69(data), nil
	case 70:
		return NewMapper70(data), nil
	default:
		return nil, fmt.Errorf("unsupported mapper: %d", mapperNumber)
	}
}