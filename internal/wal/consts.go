package wal

type entryType uint8

const (
	MaxWriteSize = 1 << 22

	entryTypeData   = entryType(0)
	entryTypeTrim   = entryType(1)
	entryTypeFooter = entryType(2)
	entryTypeMask   = 0x0F

	entryBaseSize = 8
)
