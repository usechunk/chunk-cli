// Package checksum provides file integrity verification using SHA-256 and SHA-512 checksums.
package checksum

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// Algorithm represents a supported hash algorithm.
type Algorithm string

const (
	// AlgorithmSHA256 represents the SHA-256 hash algorithm.
	AlgorithmSHA256 Algorithm = "sha256"
	// AlgorithmSHA512 represents the SHA-512 hash algorithm.
	AlgorithmSHA512 Algorithm = "sha512"
)

var (
	// ErrChecksumMismatch is returned when file checksum doesn't match the expected value.
	ErrChecksumMismatch = errors.New("checksum mismatch")
	// ErrNoChecksum is returned when no checksum is provided for verification.
	ErrNoChecksum = errors.New("no checksum provided")
	// ErrUnsupportedAlgorithm is returned when an unsupported hash algorithm is specified.
	ErrUnsupportedAlgorithm = errors.New("unsupported hash algorithm")
)

// MismatchError provides detailed information about a checksum mismatch.
type MismatchError struct {
	FilePath  string
	Algorithm Algorithm
	Expected  string
	Actual    string
}

func (e *MismatchError) Error() string {
	return fmt.Sprintf("checksum mismatch for %s: expected %s %s, got %s",
		e.FilePath, e.Algorithm, e.Expected, e.Actual)
}

func (e *MismatchError) Unwrap() error {
	return ErrChecksumMismatch
}

// Checksums holds SHA-256 and SHA-512 checksums for a file.
type Checksums struct {
	SHA256 string
	SHA512 string
}

// HasAny returns true if at least one checksum is set.
func (c *Checksums) HasAny() bool {
	return c.SHA256 != "" || c.SHA512 != ""
}

// CalculateFile calculates checksums for a file at the given path.
func CalculateFile(filePath string) (*Checksums, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer file.Close()

	return Calculate(file)
}

// Calculate calculates both SHA-256 and SHA-512 checksums for the given reader.
func Calculate(r io.Reader) (*Checksums, error) {
	sha256Hash := sha256.New()
	sha512Hash := sha512.New()

	// Use a multi-writer to calculate both hashes in a single pass
	multiWriter := io.MultiWriter(sha256Hash, sha512Hash)

	if _, err := io.Copy(multiWriter, r); err != nil {
		return nil, fmt.Errorf("failed to calculate checksums: %w", err)
	}

	return &Checksums{
		SHA256: hex.EncodeToString(sha256Hash.Sum(nil)),
		SHA512: hex.EncodeToString(sha512Hash.Sum(nil)),
	}, nil
}

// VerifyFile verifies a file's checksum against expected values.
// It will verify SHA-512 first if available, then SHA-256.
// Returns nil if verification passes, or an error if it fails.
func VerifyFile(filePath string, expected *Checksums) error {
	if expected == nil || !expected.HasAny() {
		return ErrNoChecksum
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for verification: %w", err)
	}
	defer file.Close()

	calculated, err := Calculate(file)
	if err != nil {
		return err
	}

	// Prefer SHA-512 if available
	if expected.SHA512 != "" {
		if !strings.EqualFold(calculated.SHA512, expected.SHA512) {
			return &MismatchError{
				FilePath:  filePath,
				Algorithm: AlgorithmSHA512,
				Expected:  expected.SHA512,
				Actual:    calculated.SHA512,
			}
		}
		return nil
	}

	// Fall back to SHA-256
	if expected.SHA256 != "" {
		if !strings.EqualFold(calculated.SHA256, expected.SHA256) {
			return &MismatchError{
				FilePath:  filePath,
				Algorithm: AlgorithmSHA256,
				Expected:  expected.SHA256,
				Actual:    calculated.SHA256,
			}
		}
		return nil
	}

	return nil
}

// VerifyReader verifies the checksum of data from a reader against expected values.
// This is useful when you want to verify data as it's being written.
func VerifyReader(r io.Reader, expected *Checksums) (io.Reader, *VerificationResult) {
	result := &VerificationResult{}

	if expected == nil || !expected.HasAny() {
		result.Skipped = true
		return r, result
	}

	var h hash.Hash
	if expected.SHA512 != "" {
		h = sha512.New()
		result.Algorithm = AlgorithmSHA512
		result.Expected = expected.SHA512
	} else {
		h = sha256.New()
		result.Algorithm = AlgorithmSHA256
		result.Expected = expected.SHA256
	}

	result.hash = h
	return io.TeeReader(r, h), result
}

// VerificationResult holds the result of a streaming verification.
type VerificationResult struct {
	Algorithm Algorithm
	Expected  string
	Actual    string
	Skipped   bool
	hash      hash.Hash
}

// Verify completes the verification after all data has been read.
// Returns nil if verification passes, or an error if it fails.
func (v *VerificationResult) Verify(filePath string) error {
	if v.Skipped {
		return nil
	}

	if v.hash == nil {
		return ErrNoChecksum
	}

	v.Actual = hex.EncodeToString(v.hash.Sum(nil))

	if !strings.EqualFold(v.Actual, v.Expected) {
		return &MismatchError{
			FilePath:  filePath,
			Algorithm: v.Algorithm,
			Expected:  v.Expected,
			Actual:    v.Actual,
		}
	}

	return nil
}
