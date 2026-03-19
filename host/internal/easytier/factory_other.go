//go:build !windows

package easytier

import "fmt"

func newPlatformAdapter() (Adapter, error) {
	return nil, fmt.Errorf("EasyTier FFI auto-loading is only implemented on Windows in this PoC")
}
