package backup

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Magic number for backup files: "SCTL_BKP"
var MagicNumber = [8]byte{'S', 'C', 'T', 'L', '_', 'B', 'K', 'P'}

// Current backup format version.
const FormatVersion = 1

// EncryptionMode specifies how the backup is encrypted.
type EncryptionMode string

const (
	// EncryptionModeMaster uses the vault master password.
	EncryptionModeMaster EncryptionMode = "master"
	// EncryptionModeKey uses a separate key file.
	EncryptionModeKey EncryptionMode = "key"
)

// KDFParams contains Argon2id key derivation parameters.
type KDFParams struct {
	Salt        []byte `json:"salt"`        // Base64-encoded salt
	Memory      uint32 `json:"memory"`      // Memory in KiB
	Iterations  uint32 `json:"iterations"`  // Time cost
	Parallelism uint8  `json:"parallelism"` // Threads
}

// Header contains backup file metadata.
type Header struct {
	Version        int            `json:"version"`
	CreatedAt      time.Time      `json:"created_at"`
	VaultVersion   int            `json:"vault_version"`
	EncryptionMode EncryptionMode `json:"encryption_mode"`
	KDFParams      *KDFParams     `json:"kdf_params,omitempty"` // nil if EncryptionModeKey
	IncludesAudit  bool           `json:"includes_audit"`
	SecretCount    int            `json:"secret_count"`
	ChecksumAlgo   string         `json:"checksum_algorithm"`
}

// Payload contains the encrypted backup data.
type Payload struct {
	VaultSalt []byte `json:"vault_salt"` // Original vault salt
	VaultMeta []byte `json:"vault_meta"` // vault.meta JSON
	VaultDB   []byte `json:"vault_db"`   // vault.db SQLite
	AuditLog  []byte `json:"audit_log"`  // Optional audit.jsonl
}

// WriteHeader writes the magic number and header to the writer.
func WriteHeader(w io.Writer, header *Header) error {
	// Write magic number
	if _, err := w.Write(MagicNumber[:]); err != nil {
		return fmt.Errorf("failed to write magic number: %w", err)
	}

	// Encode header to JSON
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("failed to marshal header: %w", err)
	}

	// Write header length (4 bytes, big-endian)
	headerLen := uint32(len(headerJSON))
	if err := binary.Write(w, binary.BigEndian, headerLen); err != nil {
		return fmt.Errorf("failed to write header length: %w", err)
	}

	// Write header JSON
	if _, err := w.Write(headerJSON); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	return nil
}

// ReadHeader reads and validates the magic number and header from the reader.
func ReadHeader(r io.Reader) (*Header, error) {
	// Read and verify magic number
	var magic [8]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return nil, fmt.Errorf("failed to read magic number: %w", err)
	}
	if magic != MagicNumber {
		return nil, ErrInvalidMagic
	}

	// Read header length
	var headerLen uint32
	if err := binary.Read(r, binary.BigEndian, &headerLen); err != nil {
		return nil, fmt.Errorf("failed to read header length: %w", err)
	}

	// Sanity check: header should not be larger than 1MB
	if headerLen > 1024*1024 {
		return nil, fmt.Errorf("header too large: %d bytes", headerLen)
	}

	// Read header JSON
	headerJSON := make([]byte, headerLen)
	if _, err := io.ReadFull(r, headerJSON); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Decode header
	var header Header
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("failed to unmarshal header: %w", err)
	}

	// Version check
	if header.Version > FormatVersion {
		return nil, fmt.Errorf("%w: got %d, max supported %d",
			ErrUnsupportedVersion, header.Version, FormatVersion)
	}

	return &header, nil
}

// EncodePayload encodes the payload to JSON bytes.
func EncodePayload(payload *Payload) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	return data, nil
}

// DecodePayload decodes JSON bytes to a payload.
func DecodePayload(data []byte) (*Payload, error) {
	var payload Payload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return &payload, nil
}

// HeaderBytes returns the serialized header for HMAC calculation.
func HeaderBytes(header *Header) ([]byte, error) {
	return json.Marshal(header)
}
