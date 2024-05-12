package util

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli) // Pre-calculate the table

// CalculateCRC32 calculates and returns the CRC-32 checksum of the data from the provided Reader.
func CalculateCRC32(src io.Reader) (uint32, error) {
	hasher := crc32.New(crc32cTable)

	if _, err := io.Copy(hasher, src); err != nil {
		return 0, fmt.Errorf("error calculating CRC-32: %w", err) // Wrap error
	}

	return hasher.Sum32(), nil // Return checksum and nil error on success
}

// CalculateFileCRC32 calculates and returns the CRC-32 checksum of a file.
func CalculateFileCRC32(filePath string) (uint32, error) {
	// Open file with simplified flags and permissions
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close() // Ensure file closure

	return CalculateCRC32(file)
}
