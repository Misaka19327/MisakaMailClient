//go:build windows

package syspath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// AddToUserPath adds dir to the current user's PATH environment variable
// (HKCU\Environment\Path) if it is not already present. It returns whether the
// path was added. A WM_SETTINGCHANGE broadcast notifies running processes
// (notably Explorer) so new terminals pick up the change without logging out.
func AddToUserPath(dir string) (bool, error) {
	dir = filepath.Clean(dir)

	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return false, fmt.Errorf("open HKCU\\Environment: %w", err)
	}
	defer k.Close()

	rawCur, valType, err := k.GetStringValue("Path")
	if err != nil {
		if err == registry.ErrNotExist {
			// No Path value yet; create it as REG_EXPAND_SZ (the conventional type).
			rawCur = ""
			valType = registry.EXPAND_SZ
		} else {
			return false, fmt.Errorf("read user Path: %w", err)
		}
	}

	// Compare against the EXPANDED value so an existing "%USERPROFILE%\go\bin"
	// entry is recognized as matching a literal "C:\Users\...\go\bin".
	expanded := rawCur
	if e, err := registry.ExpandString(rawCur); err == nil {
		expanded = e
	}
	if containsEntry(expanded, dir) {
		return false, nil
	}

	var newPath string
	if rawCur == "" {
		newPath = dir
	} else {
		newPath = strings.TrimRight(rawCur, ";") + ";" + dir
	}

	if valType == registry.EXPAND_SZ {
		err = k.SetExpandStringValue("Path", newPath)
	} else {
		err = k.SetStringValue("Path", newPath)
	}
	if err != nil {
		return false, fmt.Errorf("write user Path: %w", err)
	}

	broadcastSettingChange()
	// Best-effort: make the current process see the new entry too.
	if cur := os.Getenv("PATH"); cur != "" && !strings.HasSuffix(cur, ";") {
		os.Setenv("PATH", cur+";"+dir)
	} else {
		os.Setenv("PATH", cur+dir)
	}
	return true, nil
}

// containsEntry reports whether dir is already one of the ';'-separated entries
// in pathList (case-insensitive, drive letters and separators normalized).
func containsEntry(pathList, dir string) bool {
	want := normalize(dir)
	for _, p := range strings.Split(pathList, ";") {
		if normalize(p) == want {
			return true
		}
	}
	return false
}

func normalize(p string) string {
	p = strings.TrimSpace(p)
	p = filepath.Clean(p)
	return strings.ToLower(p)
}

func broadcastSettingChange() {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")
	env, err := syscall.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	const (
		hwndBroadcast   uintptr = 0xFFFF
		wmSettingChange uintptr = 0x001A
		smtoAbortIfHung uintptr = 0x0002
	)
	var result uintptr
	_, _, _ = proc.Call(
		hwndBroadcast,
		wmSettingChange,
		0,
		uintptr(unsafe.Pointer(env)),
		smtoAbortIfHung,
		1000,
		uintptr(unsafe.Pointer(&result)),
	)
}
