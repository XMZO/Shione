//go:build windows

package easytier

import (
	"fmt"
	"os"
	"path/filepath"
)

func newPlatformAdapter() (Adapter, error) {
	dllPath, err := findDLLPath()
	if err != nil {
		return nil, err
	}
	return NewDLLAdapter(dllPath)
}

func findDLLPath() (string, error) {
	candidates := make([]string, 0, 4)
	if envPath := os.Getenv("EASYTIER_FFI_DLL"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "..", "onani", "EasyTier", "target", "release", "easytier_ffi.dll"),
			filepath.Join(wd, "onani", "EasyTier", "target", "release", "easytier_ffi.dll"),
		)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "..", "onani", "EasyTier", "target", "release", "easytier_ffi.dll"),
			filepath.Join(exeDir, "easytier_ffi.dll"),
		)
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		}
	}

	return "", fmt.Errorf("could not find easytier_ffi.dll; set EASYTIER_FFI_DLL or build EasyTier FFI first")
}
