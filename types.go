package editor

import (
	"errors"
)

const (
	MagicNumber       uint32 = 0xFAB11BAF
	HeaderSize        int64  = 24 // Size of the header we care about
	LiteralEntrySize  int    = 8  // Size of each literal entry (Length + Offset)
	AlignmentBoundary uint32 = 4  // 4-byte alignment
)

type StringLiteral struct {
	Nth   int    `json:"nth"`   // Index of the string (0-based)
	Value string `json:"value"` // UTF-8 string value
}

type literalEntry struct {
	Length uint32 // Length of the string in bytes
	Offset uint32 // Offset from stringLiteralDataOffset
}

type MetadataFile struct {
	filePath                       string // Original file path
	version                        int32
	stringLiteralOffset            uint32
	stringLiteralCount             uint32 // Size in bytes, not count
	stringLiteralDataOffset        uint32
	stringLiteralDataCount         uint32 // Current size
	originalStringLiteralDataCount uint32 // Original size from file (immutable)
	originalFileSize               int64
	literals                       []literalEntry
	stringData                     []byte
	headerData                     []byte
}

var (
	ErrInvalidMagic  = errors.New("invalid magic number, not a valid metadata file")
	ErrInvalidIndex  = errors.New("invalid string index")
	ErrFileRead      = errors.New("failed to read file")
	ErrFileWrite     = errors.New("failed to write file")
	ErrInvalidFormat = errors.New("invalid file format")
)
