package checksum

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name           string
		data           string
		expectedSHA256 string
		expectedSHA512 string
	}{
		{
			name:           "empty data",
			data:           "",
			expectedSHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expectedSHA512: "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
		},
		{
			name:           "simple text",
			data:           "hello world",
			expectedSHA256: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			expectedSHA512: "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f",
		},
		{
			name:           "binary-like data",
			data:           "\x00\x01\x02\x03\x04\x05",
			expectedSHA256: "17e88db187afd62c16e5debf3e6527cd006bc012bc90b51a810cd80c2d511f43",
			expectedSHA512: "2f3831bccc94cf061bcfa5f8c23c1429d26e3bc6b76edad93d9025cb91c903af6cf9c935dc37193c04c2c66e7d9de17c358284418218afea2160147aaa912f4c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.data)
			checksums, err := Calculate(reader)
			if err != nil {
				t.Fatalf("Calculate() error = %v", err)
			}

			if checksums.SHA256 != tt.expectedSHA256 {
				t.Errorf("SHA256 = %v, want %v", checksums.SHA256, tt.expectedSHA256)
			}
			if checksums.SHA512 != tt.expectedSHA512 {
				t.Errorf("SHA512 = %v, want %v", checksums.SHA512, tt.expectedSHA512)
			}
		})
	}
}

func TestCalculateFile(t *testing.T) {
	// Create a temporary file
	tmpDir, err := os.MkdirTemp("", "checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test file content for checksum verification")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	checksums, err := CalculateFile(testFile)
	if err != nil {
		t.Fatalf("CalculateFile() error = %v", err)
	}

	if checksums.SHA256 == "" {
		t.Error("SHA256 should not be empty")
	}
	if checksums.SHA512 == "" {
		t.Error("SHA512 should not be empty")
	}

	// Verify the checksums match when calculated again
	checksums2, err := CalculateFile(testFile)
	if err != nil {
		t.Fatalf("CalculateFile() second call error = %v", err)
	}

	if checksums.SHA256 != checksums2.SHA256 {
		t.Errorf("SHA256 mismatch on second calculation: %v != %v", checksums.SHA256, checksums2.SHA256)
	}
	if checksums.SHA512 != checksums2.SHA512 {
		t.Errorf("SHA512 mismatch on second calculation: %v != %v", checksums.SHA512, checksums2.SHA512)
	}
}

func TestCalculateFileNotFound(t *testing.T) {
	_, err := CalculateFile("/non/existent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestVerifyFile(t *testing.T) {
	// Create a temporary file
	tmpDir, err := os.MkdirTemp("", "checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("hello world")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		checksums   *Checksums
		wantErr     bool
		errType     error
		description string
	}{
		{
			name: "valid SHA256",
			checksums: &Checksums{
				SHA256: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			},
			wantErr:     false,
			description: "should pass with correct SHA256",
		},
		{
			name: "valid SHA512",
			checksums: &Checksums{
				SHA512: "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f",
			},
			wantErr:     false,
			description: "should pass with correct SHA512",
		},
		{
			name: "valid both checksums",
			checksums: &Checksums{
				SHA256: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
				SHA512: "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f",
			},
			wantErr:     false,
			description: "should prefer SHA512 when both provided",
		},
		{
			name: "invalid SHA256",
			checksums: &Checksums{
				SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
			},
			wantErr:     true,
			errType:     ErrChecksumMismatch,
			description: "should fail with wrong SHA256",
		},
		{
			name: "invalid SHA512",
			checksums: &Checksums{
				SHA512: "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			},
			wantErr:     true,
			errType:     ErrChecksumMismatch,
			description: "should fail with wrong SHA512",
		},
		{
			name:        "nil checksums",
			checksums:   nil,
			wantErr:     true,
			errType:     ErrNoChecksum,
			description: "should fail with nil checksums",
		},
		{
			name:        "empty checksums",
			checksums:   &Checksums{},
			wantErr:     true,
			errType:     ErrNoChecksum,
			description: "should fail with empty checksums",
		},
		{
			name: "case insensitive SHA256",
			checksums: &Checksums{
				SHA256: "B94D27B9934D3E08A52E52D7DA7DABFAC484EFE37A5380EE9088F7ACE2EFCDE9",
			},
			wantErr:     false,
			description: "should be case insensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyFile(testFile, tt.checksums)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyFile() error = %v, wantErr %v (%s)", err, tt.wantErr, tt.description)
			}
			if tt.errType != nil && err != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("VerifyFile() error type = %T, want %T", err, tt.errType)
				}
			}
		})
	}
}

func TestVerifyFileNotFound(t *testing.T) {
	checksums := &Checksums{
		SHA256: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
	}
	err := VerifyFile("/non/existent/file.txt", checksums)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestMismatchError(t *testing.T) {
	err := &MismatchError{
		FilePath:  "/path/to/file.jar",
		Algorithm: AlgorithmSHA256,
		Expected:  "expected123",
		Actual:    "actual456",
	}

	// Test Error() method
	errStr := err.Error()
	if !strings.Contains(errStr, "/path/to/file.jar") {
		t.Error("Error message should contain file path")
	}
	if !strings.Contains(errStr, "sha256") {
		t.Error("Error message should contain algorithm")
	}
	if !strings.Contains(errStr, "expected123") {
		t.Error("Error message should contain expected checksum")
	}
	if !strings.Contains(errStr, "actual456") {
		t.Error("Error message should contain actual checksum")
	}

	// Test Unwrap() method
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Error("MismatchError should unwrap to ErrChecksumMismatch")
	}
}

func TestChecksums_HasAny(t *testing.T) {
	tests := []struct {
		name      string
		checksums *Checksums
		want      bool
	}{
		{
			name:      "empty checksums",
			checksums: &Checksums{},
			want:      false,
		},
		{
			name: "only SHA256",
			checksums: &Checksums{
				SHA256: "abc123",
			},
			want: true,
		},
		{
			name: "only SHA512",
			checksums: &Checksums{
				SHA512: "def456",
			},
			want: true,
		},
		{
			name: "both checksums",
			checksums: &Checksums{
				SHA256: "abc123",
				SHA512: "def456",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.checksums.HasAny(); got != tt.want {
				t.Errorf("Checksums.HasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyReader(t *testing.T) {
	data := []byte("hello world")
	expectedSHA256 := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	checksums := &Checksums{
		SHA256: expectedSHA256,
	}

	reader, result := VerifyReader(bytes.NewReader(data), checksums)

	// Read all data
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(reader)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	// Verify
	err = result.Verify("test.txt")
	if err != nil {
		t.Errorf("VerifyReader() verification failed: %v", err)
	}

	// Check that data was read correctly
	if !bytes.Equal(buf.Bytes(), data) {
		t.Error("Data was corrupted during verification")
	}
}

func TestVerifyReaderMismatch(t *testing.T) {
	data := []byte("hello world")
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

	checksums := &Checksums{
		SHA256: wrongChecksum,
	}

	reader, result := VerifyReader(bytes.NewReader(data), checksums)

	// Read all data
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(reader)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	// Verify should fail
	err = result.Verify("test.txt")
	if err == nil {
		t.Error("Expected verification to fail")
	}
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Errorf("Expected ErrChecksumMismatch, got %v", err)
	}
}

func TestVerifyReaderSkipped(t *testing.T) {
	data := []byte("hello world")

	reader, result := VerifyReader(bytes.NewReader(data), nil)

	// Read all data
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(reader)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	// Verify should be skipped
	err = result.Verify("test.txt")
	if err != nil {
		t.Errorf("Expected no error when skipped, got %v", err)
	}

	if !result.Skipped {
		t.Error("Expected result.Skipped to be true")
	}
}

func TestAlgorithmConstants(t *testing.T) {
	if AlgorithmSHA256 != "sha256" {
		t.Errorf("AlgorithmSHA256 = %v, want sha256", AlgorithmSHA256)
	}
	if AlgorithmSHA512 != "sha512" {
		t.Errorf("AlgorithmSHA512 = %v, want sha512", AlgorithmSHA512)
	}
}
