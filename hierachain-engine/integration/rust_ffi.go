package integration

/*
#cgo CFLAGS: -I${SRCDIR}/../include
#cgo windows LDFLAGS: -L${SRCDIR}/../../target/release -lhierachain_consensus -lws2_32 -luserenv -lbcrypt -lntdll
#cgo linux LDFLAGS: -L${SRCDIR}/../../target/release -lhierachain_consensus -lm -ldl -lpthread
#cgo darwin LDFLAGS: -L${SRCDIR}/../../target/release -lhierachain_consensus -framework Security -framework CoreFoundation

#include <stdlib.h>
#include <stdint.h>
#include <string.h>

// FFI function declarations from Rust
extern int32_t ffi_calculate_merkle_root(const char* events_json, char* result, size_t result_len);
extern int32_t ffi_calculate_block_hash(const char* block_json, char* result, size_t result_len);
extern int32_t ffi_bulk_validate_transactions(const char* transactions_json);
extern int32_t ffi_process_arrow_batch(const uint8_t* arrow_ipc, size_t arrow_ipc_len,
                                        uint8_t* result, size_t result_capacity, size_t* result_len);
extern int32_t ffi_get_version(char* result, size_t result_len);
*/
import "C"

import (
	"errors"
	"unsafe"
)

// FFI error codes (must match Rust ffi.rs)
const (
	FFISuccess          = 0
	FFIErrorNullPointer = -1
	FFIErrorInvalidUTF8 = -2
	FFIErrorJSONParse   = -3
	FFIErrorBufferSmall = -4
	FFIErrorInternal    = -5
)

// MaxFFIInputSize is the maximum allowed input size for FFI calls (100MB).
// This prevents memory exhaustion from oversized inputs.
const MaxFFIInputSize = 100 * 1024 * 1024 // 100MB

// ErrFFIInputTooLarge is returned when FFI input exceeds MaxFFIInputSize.
var ErrFFIInputTooLarge = errors.New("ffi input size exceeds maximum allowed")

// ffiCodeToError converts FFI error code to Go error
func ffiCodeToError(code C.int32_t) error {
	switch code {
	case FFISuccess:
		return nil
	case FFIErrorNullPointer:
		return errors.New("rust ffi: null pointer")
	case FFIErrorInvalidUTF8:
		return errors.New("rust ffi: invalid UTF-8")
	case FFIErrorJSONParse:
		return errors.New("rust ffi: JSON parse error")
	case FFIErrorBufferSmall:
		return errors.New("rust ffi: buffer too small")
	case FFIErrorInternal:
		return errors.New("rust ffi: internal error")
	default:
		return errors.New("rust ffi: unknown error")
	}
}

// RustMerkleRoot calculates the Merkle root using Rust implementation.
// Input: JSON string of events array, e.g., `[{"entity_id":"a","event":"b"},...]`
// Returns: 64-character hex hash string
func RustMerkleRoot(eventsJSON []byte) (string, error) {
	if len(eventsJSON) == 0 {
		return "", errors.New("empty events JSON")
	}
	if len(eventsJSON) > MaxFFIInputSize {
		return "", ErrFFIInputTooLarge
	}

	cJSON := C.CString(string(eventsJSON))
	defer C.free(unsafe.Pointer(cJSON))

	resultBuf := make([]byte, 128)
	code := C.ffi_calculate_merkle_root(
		cJSON,
		(*C.char)(unsafe.Pointer(&resultBuf[0])),
		C.size_t(len(resultBuf)),
	)

	if err := ffiCodeToError(code); err != nil {
		return "", err
	}

	// Find null terminator
	for i, b := range resultBuf {
		if b == 0 {
			return string(resultBuf[:i]), nil
		}
	}
	return string(resultBuf), nil
}

// RustBlockHash calculates the block hash using Rust implementation.
// Input: JSON string of block data
// Returns: 64-character hex hash string
func RustBlockHash(blockJSON []byte) (string, error) {
	if len(blockJSON) == 0 {
		return "", errors.New("empty block JSON")
	}
	if len(blockJSON) > MaxFFIInputSize {
		return "", ErrFFIInputTooLarge
	}

	cJSON := C.CString(string(blockJSON))
	defer C.free(unsafe.Pointer(cJSON))

	resultBuf := make([]byte, 128)
	code := C.ffi_calculate_block_hash(
		cJSON,
		(*C.char)(unsafe.Pointer(&resultBuf[0])),
		C.size_t(len(resultBuf)),
	)

	if err := ffiCodeToError(code); err != nil {
		return "", err
	}

	// Find null terminator
	for i, b := range resultBuf {
		if b == 0 {
			return string(resultBuf[:i]), nil
		}
	}
	return string(resultBuf), nil
}

// RustValidateTransactions validates a batch of transactions using Rust.
// Input: JSON string of transactions array
// Returns: true if all valid, false otherwise
func RustValidateTransactions(transactionsJSON []byte) (bool, error) {
	if len(transactionsJSON) == 0 {
		return false, errors.New("empty transactions JSON")
	}
	if len(transactionsJSON) > MaxFFIInputSize {
		return false, ErrFFIInputTooLarge
	}

	cJSON := C.CString(string(transactionsJSON))
	defer C.free(unsafe.Pointer(cJSON))

	result := C.ffi_bulk_validate_transactions(cJSON)

	if result < 0 {
		return false, ffiCodeToError(result)
	}

	return result == 1, nil
}

// RustProcessArrowBatch processes Arrow IPC data through Rust.
// Used for validation or transformation of Arrow batches.
func RustProcessArrowBatch(arrowIPC []byte) ([]byte, error) {
	if len(arrowIPC) == 0 {
		return nil, errors.New("empty arrow IPC data")
	}
	if len(arrowIPC) > MaxFFIInputSize {
		return nil, ErrFFIInputTooLarge
	}

	// Allocate result buffer (same size or larger)
	resultCapacity := len(arrowIPC) * 2
	if resultCapacity < 1024 {
		resultCapacity = 1024
	}
	resultBuf := make([]byte, resultCapacity)
	var resultLen C.size_t

	code := C.ffi_process_arrow_batch(
		(*C.uint8_t)(unsafe.Pointer(&arrowIPC[0])),
		C.size_t(len(arrowIPC)),
		(*C.uint8_t)(unsafe.Pointer(&resultBuf[0])),
		C.size_t(resultCapacity),
		&resultLen,
	)

	if err := ffiCodeToError(code); err != nil {
		return nil, err
	}

	return resultBuf[:resultLen], nil
}

// RustVersion returns the version of the Rust library.
func RustVersion() (string, error) {
	resultBuf := make([]byte, 64)
	code := C.ffi_get_version(
		(*C.char)(unsafe.Pointer(&resultBuf[0])),
		C.size_t(len(resultBuf)),
	)

	if err := ffiCodeToError(code); err != nil {
		return "", err
	}

	// Find null terminator
	for i, b := range resultBuf {
		if b == 0 {
			return string(resultBuf[:i]), nil
		}
	}
	return string(resultBuf), nil
}

// IsRustAvailable checks if the Rust library is properly linked.
func IsRustAvailable() bool {
	_, err := RustVersion()
	return err == nil
}
