package editor

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// load a global-metadata.dat file and parse its structure.
func LoadMetadata(filePath string) (*MetadataFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFileRead, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get file info: %v", ErrFileRead, err)
	}

	mf := &MetadataFile{
		filePath:         filePath,
		originalFileSize: fileInfo.Size(),
	}

	// Read and validate header
	if err := mf.readHeader(file); err != nil {
		return nil, err
	}

	// Read entire file
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("%w: failed to seek: %v", ErrFileRead, err)
	}
	mf.headerData, err = io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read file: %v", ErrFileRead, err)
	}

	// Read string literal entries
	if err := mf.readLiteralEntries(file); err != nil {
		return nil, err
	}

	// Read string data
	if err := mf.readStringData(file); err != nil {
		return nil, err
	}

	return mf, nil
}

// read and validate the file header.
func (mf *MetadataFile) readHeader(file *os.File) error {
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("%w: failed to seek to start: %v", ErrFileRead, err)
	}

	// Read magic number
	var magic uint32
	if err := binary.Read(file, binary.LittleEndian, &magic); err != nil {
		return fmt.Errorf("%w: failed to read magic: %v", ErrFileRead, err)
	}
	if magic != MagicNumber {
		return fmt.Errorf("%w: got 0x%X, expected 0x%X", ErrInvalidMagic, magic, MagicNumber)
	}

	// Read version
	if err := binary.Read(file, binary.LittleEndian, &mf.version); err != nil {
		return fmt.Errorf("%w: failed to read version: %v", ErrFileRead, err)
	}

	// Read string literal metadata
	if err := binary.Read(file, binary.LittleEndian, &mf.stringLiteralOffset); err != nil {
		return fmt.Errorf("%w: failed to read stringLiteralOffset: %v", ErrFileRead, err)
	}
	if err := binary.Read(file, binary.LittleEndian, &mf.stringLiteralCount); err != nil {
		return fmt.Errorf("%w: failed to read stringLiteralCount: %v", ErrFileRead, err)
	}
	if err := binary.Read(file, binary.LittleEndian, &mf.stringLiteralDataOffset); err != nil {
		return fmt.Errorf("%w: failed to read stringLiteralDataOffset: %v", ErrFileRead, err)
	}
	if err := binary.Read(file, binary.LittleEndian, &mf.stringLiteralDataCount); err != nil {
		return fmt.Errorf("%w: failed to read stringLiteralDataCount: %v", ErrFileRead, err)
	}

	// Save original data count
	mf.originalStringLiteralDataCount = mf.stringLiteralDataCount

	return nil
}

// read all string literal entries from the file.
func (mf *MetadataFile) readLiteralEntries(file *os.File) error {
	if _, err := file.Seek(int64(mf.stringLiteralOffset), 0); err != nil {
		return fmt.Errorf("%w: failed to seek to literal offset: %v", ErrFileRead, err)
	}

	entryCount := int(mf.stringLiteralCount) / LiteralEntrySize
	mf.literals = make([]literalEntry, entryCount)

	for i := 0; i < entryCount; i++ {
		if err := binary.Read(file, binary.LittleEndian, &mf.literals[i].Length); err != nil {
			return fmt.Errorf("%w: failed to read literal length at index %d: %v", ErrFileRead, i, err)
		}
		if err := binary.Read(file, binary.LittleEndian, &mf.literals[i].Offset); err != nil {
			return fmt.Errorf("%w: failed to read literal offset at index %d: %v", ErrFileRead, i, err)
		}
	}

	return nil
}

// read the entire string data section.
func (mf *MetadataFile) readStringData(file *os.File) error {
	if _, err := file.Seek(int64(mf.stringLiteralDataOffset), 0); err != nil {
		return fmt.Errorf("%w: failed to seek to string data: %v", ErrFileRead, err)
	}

	// Header's data count may include alignment padding that doesn't
	// physically exist in the file (C# editor quirk). Cap to available bytes.
	readSize := int64(mf.stringLiteralDataCount)
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("%w: failed to stat file: %v", ErrFileRead, err)
	}
	available := fileInfo.Size() - int64(mf.stringLiteralDataOffset)
	if available < 0 {
		available = 0
	}
	if readSize > available {
		readSize = available
	}

	mf.stringData = make([]byte, readSize)
	if _, err := io.ReadFull(file, mf.stringData); err != nil {
		return fmt.Errorf("%w: failed to read string data: %v", ErrFileRead, err)
	}

	return nil
}

// return the metadata version.
func (mf *MetadataFile) GetVersion() int32 {
	return mf.version
}

// return the total number of strings in the metadata.
func (mf *MetadataFile) GetStringCount() int {
	return len(mf.literals)
}

// return the original file size in bytes.
func (mf *MetadataFile) GetFileSize() int64 {
	return mf.originalFileSize
}

// return all string literals in the metadata file.
func (mf *MetadataFile) ListStrings() []StringLiteral {
	result := make([]StringLiteral, len(mf.literals))

	for i, entry := range mf.literals {
		start := entry.Offset
		end := entry.Offset + entry.Length

		// Bounds check
		if end > uint32(len(mf.stringData)) {
			result[i] = StringLiteral{
				Nth:   i,
				Value: "[ERROR: Invalid offset/length]",
			}
			continue
		}

		result[i] = StringLiteral{
			Nth:   i,
			Value: string(mf.stringData[start:end]),
		}
	}

	return result
}

// return a single string by its index.
func (mf *MetadataFile) GetString(nth int) (StringLiteral, error) {
	if nth < 0 || nth >= len(mf.literals) {
		return StringLiteral{}, fmt.Errorf("%w: index %d out of range [0, %d)", ErrInvalidIndex, nth, len(mf.literals))
	}

	entry := mf.literals[nth]
	start := entry.Offset
	end := entry.Offset + entry.Length

	if end > uint32(len(mf.stringData)) {
		return StringLiteral{}, fmt.Errorf("%w: string data corrupted at index %d", ErrInvalidFormat, nth)
	}

	return StringLiteral{
		Nth:   nth,
		Value: string(mf.stringData[start:end]),
	}, nil
}

// searches for strings matching the given pattern.
// If useRegex is true, pattern is treated as a regular expression.
// Otherwise, it performs a case-insensitive substring search.
func (mf *MetadataFile) SearchStrings(pattern string, useRegex bool) ([]StringLiteral, error) {
	var matcher func(string) bool

	if useRegex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %v", err)
		}
		matcher = re.MatchString
	} else {
		lowerPattern := strings.ToLower(pattern)
		matcher = func(s string) bool {
			return strings.Contains(strings.ToLower(s), lowerPattern)
		}
	}

	var results []StringLiteral
	allStrings := mf.ListStrings()

	for _, str := range allStrings {
		if matcher(str.Value) {
			results = append(results, str)
		}
	}

	return results, nil
}

// modifies a string at the given index.
func (mf *MetadataFile) ModifyString(nth int, newValue string) error {
	if nth < 0 || nth >= len(mf.literals) {
		return fmt.Errorf("%w: index %d out of range [0, %d)", ErrInvalidIndex, nth, len(mf.literals))
	}

	newBytes := []byte(newValue)

	// Rebuild the string data section
	newStringData := make([]byte, 0, len(mf.stringData))

	for i, entry := range mf.literals {
		start := entry.Offset
		end := entry.Offset + entry.Length

		if i == nth {
			// Update the entry with new data
			mf.literals[i].Offset = uint32(len(newStringData))
			mf.literals[i].Length = uint32(len(newBytes))
			newStringData = append(newStringData, newBytes...)
		} else {
			// Update offset for this entry
			mf.literals[i].Offset = uint32(len(newStringData))
			// Copy existing data
			if end <= uint32(len(mf.stringData)) {
				newStringData = append(newStringData, mf.stringData[start:end]...)
			}
		}
	}

	mf.stringData = newStringData
	mf.stringLiteralDataCount = uint32(len(newStringData))

	return nil
}

// writes the modified metadata to a new file.
func (mf *MetadataFile) Save(outputPath string) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("%w: failed to create output file: %v", ErrFileWrite, err)
	}
	defer outFile.Close()

	// Write the entire original file first
	if _, err := outFile.Write(mf.headerData); err != nil {
		return fmt.Errorf("%w: failed to write original data: %v", ErrFileWrite, err)
	}

	// Calculate new data size with alignment
	rawDataSize := uint32(len(mf.stringData))

	// Calculate alignment based on original offset + data size
	totalSizeAtOriginalOffset := mf.stringLiteralDataOffset + rawDataSize
	remainder := totalSizeAtOriginalOffset % AlignmentBoundary
	var alignedDataSize uint32
	if remainder != 0 {
		alignedDataSize = rawDataSize + (AlignmentBoundary - remainder)
	} else {
		alignedDataSize = rawDataSize
	}

	// Determine if we need to move the data section
	newDataOffset := mf.stringLiteralDataOffset
	dataSectionMoved := false

	if alignedDataSize > mf.originalStringLiteralDataCount {
		endOfDataSection := mf.stringLiteralDataOffset + mf.originalStringLiteralDataCount
		if int64(endOfDataSection) < mf.originalFileSize {
			// Move to end of file
			newDataOffset = uint32(mf.originalFileSize)
			dataSectionMoved = true
		}
	}

	// Update literal entries with new offsets and lengths
	if _, err := outFile.Seek(int64(mf.stringLiteralOffset), 0); err != nil {
		return fmt.Errorf("%w: failed to seek to literal section: %v", ErrFileWrite, err)
	}

	for _, entry := range mf.literals {
		if err := binary.Write(outFile, binary.LittleEndian, entry.Length); err != nil {
			return fmt.Errorf("%w: failed to write literal length: %v", ErrFileWrite, err)
		}
		if err := binary.Write(outFile, binary.LittleEndian, entry.Offset); err != nil {
			return fmt.Errorf("%w: failed to write literal offset: %v", ErrFileWrite, err)
		}
	}

	// Write string data section
	if _, err := outFile.Seek(int64(newDataOffset), 0); err != nil {
		return fmt.Errorf("%w: failed to seek to string data section: %v", ErrFileWrite, err)
	}

	if _, err := outFile.Write(mf.stringData); err != nil {
		return fmt.Errorf("%w: failed to write string data: %v", ErrFileWrite, err)
	}

	// Write padding bytes to make file match header's aligned count
	paddingSize := alignedDataSize - rawDataSize
	if paddingSize > 0 {
		padding := make([]byte, paddingSize)
		if _, err := outFile.Write(padding); err != nil {
			return fmt.Errorf("%w: failed to write padding: %v", ErrFileWrite, err)
		}
	}

	// Truncate the file
	// If data section didn't move, keep original file size (preserve trailing data)
	if dataSectionMoved {
		finalSize := int64(newDataOffset) + int64(alignedDataSize)
		if err := outFile.Truncate(finalSize); err != nil {
			return fmt.Errorf("%w: failed to truncate file: %v", ErrFileWrite, err)
		}
	}

	// Update header with new data offset and count
	if _, err := outFile.Seek(16, 0); err != nil { // Position of stringLiteralDataOffset
		return fmt.Errorf("%w: failed to seek to header: %v", ErrFileWrite, err)
	}

	if err := binary.Write(outFile, binary.LittleEndian, newDataOffset); err != nil {
		return fmt.Errorf("%w: failed to write data offset: %v", ErrFileWrite, err)
	}
	if err := binary.Write(outFile, binary.LittleEndian, alignedDataSize); err != nil {
		return fmt.Errorf("%w: failed to write data count: %v", ErrFileWrite, err)
	}

	return nil
}

// exports all string literals to a JSON file
func (mf *MetadataFile) ExportJSON(outputPath string) error {
	strings := mf.ListStrings()

	data, err := json.MarshalIndent(strings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("%w: failed to write JSON file: %v", ErrFileWrite, err)
	}

	return nil
}
