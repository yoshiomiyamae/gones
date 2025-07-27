package mapper

import "fmt"

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
	default:
		return nil, fmt.Errorf("unsupported mapper: %d", mapperNumber)
	}
}