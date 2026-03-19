//go:build windows

package easytier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type dllAPI struct {
	dll                     *windows.DLL
	parseConfig             *windows.Proc
	runNetworkInstance      *windows.Proc
	stopAllNetworkInstances *windows.Proc
	collectNetworkInfosJSON *windows.Proc
	getErrorMsg             *windows.Proc
	freeString              *windows.Proc
}

type DLLAdapter struct {
	mu     sync.RWMutex
	api    *dllAPI
	status Status
}

func NewDLLAdapter(dllPath string) (*DLLAdapter, error) {
	absPath, err := filepath.Abs(dllPath)
	if err != nil {
		return nil, fmt.Errorf("resolve easytier DLL path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("stat easytier DLL: %w", err)
	}

	if err := prepareRuntimeEnvironment(absPath); err != nil {
		return nil, err
	}

	dll, err := loadDLL(absPath)
	if err != nil {
		return nil, fmt.Errorf("load easytier DLL: %w", err)
	}

	find := func(name string) (*windows.Proc, error) {
		proc, err := dll.FindProc(name)
		if err != nil {
			return nil, fmt.Errorf("find easytier proc %s: %w", name, err)
		}
		return proc, nil
	}

	parseConfig, err := find("parse_config")
	if err != nil {
		return nil, err
	}
	runNetworkInstance, err := find("run_network_instance")
	if err != nil {
		return nil, err
	}
	stopAllNetworkInstances, err := find("stop_all_network_instances")
	if err != nil {
		return nil, err
	}
	collectNetworkInfosJSON, err := find("collect_network_infos_json")
	if err != nil {
		return nil, err
	}
	getErrorMsg, err := find("get_error_msg")
	if err != nil {
		return nil, err
	}
	freeString, err := find("free_string")
	if err != nil {
		return nil, err
	}

	return &DLLAdapter{
		api: &dllAPI{
			dll:                     dll,
			parseConfig:             parseConfig,
			runNetworkInstance:      runNetworkInstance,
			stopAllNetworkInstances: stopAllNetworkInstances,
			collectNetworkInfosJSON: collectNetworkInfosJSON,
			getErrorMsg:             getErrorMsg,
			freeString:              freeString,
		},
		status: Status{
			Supported: true,
			Loaded:    true,
			State:     "idle",
			DLLPath:   absPath,
		},
	}, nil
}

func (a *DLLAdapter) Start(configPath string) error {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("resolve EasyTier config path: %w", err)
	}
	content, err := os.ReadFile(absConfigPath)
	if err != nil {
		return fmt.Errorf("read EasyTier config: %w", err)
	}

	a.mu.Lock()
	if a.status.State == "running" {
		a.mu.Unlock()
		return fmt.Errorf("EasyTier is already running")
	}
	a.mu.Unlock()

	if err := a.callCStringFunc(a.api.parseConfig, content); err != nil {
		a.setError(err)
		return err
	}
	if err := a.callCStringFunc(a.api.runNetworkInstance, content); err != nil {
		a.setError(err)
		return err
	}

	a.mu.Lock()
	a.status.State = "running"
	a.status.ConfigPath = absConfigPath
	a.status.StartedAt = time.Now()
	a.status.LastError = ""
	a.mu.Unlock()

	_ = a.refreshInfos()
	return nil
}

func (a *DLLAdapter) Stop() error {
	if err := a.callNoArgFunc(a.api.stopAllNetworkInstances); err != nil {
		a.setError(err)
		return err
	}

	a.mu.Lock()
	a.status.State = "idle"
	a.status.ConfigPath = ""
	a.status.StartedAt = time.Time{}
	a.status.LastError = ""
	a.status.Infos = nil
	a.mu.Unlock()
	return nil
}

func (a *DLLAdapter) Status() Status {
	_ = a.refreshInfos()

	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *DLLAdapter) refreshInfos() error {
	raw, err := a.callStringOutFunc(a.api.collectNetworkInfosJSON)
	if err != nil {
		a.setError(err)
		return err
	}

	a.mu.Lock()
	a.status.Infos = json.RawMessage(raw)
	a.mu.Unlock()
	return nil
}

func (a *DLLAdapter) setError(err error) {
	if err == nil {
		return
	}
	a.mu.Lock()
	a.status.LastError = err.Error()
	a.mu.Unlock()
}

func (a *DLLAdapter) callNoArgFunc(proc *windows.Proc) error {
	r1, _, _ := proc.Call()
	if int32(r1) < 0 {
		return a.lastError()
	}
	return nil
}

func (a *DLLAdapter) callCStringFunc(proc *windows.Proc, content []byte) error {
	cstr := make([]byte, len(content)+1)
	copy(cstr, content)
	r1, _, _ := proc.Call(uintptr(unsafe.Pointer(&cstr[0])))
	if int32(r1) < 0 {
		return a.lastError()
	}
	return nil
}

func (a *DLLAdapter) callStringOutFunc(proc *windows.Proc) ([]byte, error) {
	var out uintptr
	r1, _, _ := proc.Call(uintptr(unsafe.Pointer(&out)))
	if int32(r1) < 0 {
		return nil, a.lastError()
	}
	if out == 0 {
		return []byte("{}"), nil
	}

	defer a.api.freeString.Call(out)
	return cStringBytes(out), nil
}

func (a *DLLAdapter) lastError() error {
	var out uintptr
	a.api.getErrorMsg.Call(uintptr(unsafe.Pointer(&out)))
	if out == 0 {
		return fmt.Errorf("EasyTier FFI call failed")
	}
	defer a.api.freeString.Call(out)
	return fmt.Errorf("%s", cStringBytes(out))
}

func cStringBytes(ptr uintptr) []byte {
	if ptr == 0 {
		return nil
	}

	bytes := make([]byte, 0, 256)
	for offset := uintptr(0); ; offset++ {
		b := *(*byte)(unsafe.Pointer(ptr + offset))
		if b == 0 {
			break
		}
		bytes = append(bytes, b)
	}
	return bytes
}

func loadDLL(path string) (*windows.DLL, error) {
	handle, err := windows.LoadLibraryEx(path, 0, windows.LOAD_WITH_ALTERED_SEARCH_PATH)
	if err != nil {
		return nil, err
	}
	return &windows.DLL{Name: path, Handle: handle}, nil
}

func prepareRuntimeEnvironment(dllPath string) error {
	dirs := runtimeDirCandidates(dllPath)
	if len(dirs) == 0 {
		return fmt.Errorf("no EasyTier runtime directories found for %s", dllPath)
	}
	prependPATH(dirs...)
	return nil
}

func runtimeDirCandidates(dllPath string) []string {
	candidates := []string{filepath.Dir(dllPath)}

	if archDir := easytierRuntimeArch(); archDir != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(dllPath), "..", "..", "easytier", "third_party", archDir))
	}

	seen := map[string]struct{}{}
	dirs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[abs] = struct{}{}
		dirs = append(dirs, abs)
	}
	return dirs
}

func easytierRuntimeArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	case "386":
		return "i686"
	default:
		return ""
	}
}

func prependPATH(dirs ...string) {
	existing := filepath.SplitList(os.Getenv("PATH"))
	combined := make([]string, 0, len(dirs)+len(existing))

	for _, dir := range dirs {
		if !containsPath(combined, dir) && !containsPath(existing, dir) {
			combined = append(combined, dir)
		}
	}

	combined = append(combined, existing...)
	_ = os.Setenv("PATH", strings.Join(combined, string(filepath.ListSeparator)))
}

func containsPath(paths []string, target string) bool {
	for _, path := range paths {
		if strings.EqualFold(path, target) {
			return true
		}
	}
	return false
}
