//go:build windows

package vault

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

func LocalProtectionAvailable() bool { return true }

func protectLocal(plaintext, entropy []byte) ([]byte, error) {
	protected, err := cryptProtectData(plaintext, entropy, true)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLocalProtectionFailure, err)
	}
	return protected, nil
}

func unprotectLocal(ciphertext, entropy []byte) ([]byte, error) {
	plaintext, err := cryptProtectData(ciphertext, entropy, false)
	if err == nil {
		return plaintext, nil
	}
	if errors.Is(err, windows.ERROR_INVALID_DATA) {
		return nil, ErrAuthentication
	}
	return nil, fmt.Errorf("%w: %v", ErrLocalProtectionFailure, err)
}

func cryptProtectData(input, entropy []byte, protect bool) ([]byte, error) {
	maximumInput := maxDPAPIBlob
	if protect {
		maximumInput = maxSecretBytes
	}
	if len(input) == 0 || len(input) > maximumInput {
		return nil, fmt.Errorf("%w: invalid DPAPI input size", ErrInvalidVault)
	}
	inputBlob := dataBlob(input)
	entropyBlob := dataBlob(entropy)
	var output windows.DataBlob
	var err error
	if protect {
		description, conversionErr := windows.UTF16PtrFromString("Entropy wallet vault")
		if conversionErr != nil {
			return nil, fmt.Errorf("prepare DPAPI description: %w", conversionErr)
		}
		err = windows.CryptProtectData(
			&inputBlob,
			description,
			&entropyBlob,
			0,
			nil,
			windows.CRYPTPROTECT_UI_FORBIDDEN,
			&output,
		)
		runtime.KeepAlive(description)
	} else {
		err = windows.CryptUnprotectData(
			&inputBlob,
			nil,
			&entropyBlob,
			0,
			nil,
			windows.CRYPTPROTECT_UI_FORBIDDEN,
			&output,
		)
	}
	runtime.KeepAlive(input)
	runtime.KeepAlive(entropy)
	if err != nil {
		return nil, errors.Join(err, releaseDPAPIOutput(&output))
	}
	maximumOutput := uint32(maxSecretBytes)
	if protect {
		maximumOutput = uint32(maxDPAPIBlob)
	}
	if output.Data == nil || output.Size == 0 || output.Size > maximumOutput {
		return nil, errors.Join(
			fmt.Errorf("%w: invalid DPAPI output size", ErrInvalidVault),
			releaseDPAPIOutput(&output),
		)
	}
	native := unsafe.Slice(output.Data, int(output.Size))
	result := append([]byte(nil), native...)
	freeErr := releaseDPAPIOutput(&output)
	if freeErr != nil {
		clear(result)
		return nil, fmt.Errorf("release DPAPI output: %w", freeErr)
	}
	return result, nil
}

func releaseDPAPIOutput(output *windows.DataBlob) error {
	if output == nil || output.Data == nil {
		return nil
	}
	if output.Size > 0 && uint64(output.Size) <= uint64(^uint(0)>>1) {
		clear(unsafe.Slice(output.Data, int(output.Size)))
	}
	_, err := windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(output.Data))))
	output.Data = nil
	output.Size = 0
	return err
}

func dataBlob(value []byte) windows.DataBlob {
	if len(value) == 0 {
		return windows.DataBlob{}
	}
	return windows.DataBlob{Size: uint32(len(value)), Data: &value[0]}
}
