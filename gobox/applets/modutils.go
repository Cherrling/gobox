package applets

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"syscall"
	"unsafe"
)

func init() {
	Register("lsmod", AppletFunc(lsmodMain))
	Register("insmod", AppletFunc(insmodMain))
	Register("rmmod", AppletFunc(rmmodMain))
	Register("modprobe", AppletFunc(modprobeMain))
	Register("modinfo", AppletFunc(modinfoMain))
	Register("depmod", AppletFunc(depmodMain))
}

func lsmodMain(args []string) int {
	data, err := os.ReadFile("/proc/modules")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: lsmod: cannot read /proc/modules")
		return 1
	}
	os.Stdout.Write(data)
	return 0
}

func insmodMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: insmod: missing module")
		return 1
	}

	path := args[1]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: insmod: %s: %v\n", path, err)
		return 1
	}

	// Use init_module syscall
	_, _, errno := syscall.Syscall(syscall.SYS_INIT_MODULE,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		0)
	if errno != 0 {
		fmt.Fprintf(os.Stderr, "gobox: insmod: %s: %v\n", path, errno)
		return 1
	}
	return 0
}

func rmmodMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: rmmod: missing module")
		return 1
	}

	moduleName := args[1]
	// Use delete_module syscall
	nameBytes := append([]byte(moduleName), 0)
	_, _, errno := syscall.Syscall(syscall.SYS_DELETE_MODULE,
		uintptr(unsafe.Pointer(&nameBytes[0])),
		syscall.O_NONBLOCK,
		0)
	if errno != 0 {
		fmt.Fprintf(os.Stderr, "gobox: rmmod: %s: %v\n", moduleName, errno)
		return 1
	}
	return 0
}

func modprobeMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: modprobe: missing module")
		return 1
	}

	module := args[1]
	// Try to load from /lib/modules
	release, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: modprobe: %v\n", err)
		return 1
	}

	kver := strings.TrimSpace(string(release))
	modDir := filepath.Join("/lib/modules", kver)

	// Search for .ko files
	var modPath string
	filepath.Walk(modDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ko") {
			base := filepath.Base(path)
			name := strings.TrimSuffix(base, ".ko")
			if name == module || strings.HasSuffix(name, "/"+module) {
				modPath = path
			}
		}
		return nil
	})

	if modPath == "" {
		fmt.Fprintf(os.Stderr, "gobox: modprobe: module '%s' not found\n", module)
		return 1
	}

	return insmodMain([]string{"insmod", modPath})
}

func modinfoMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: modinfo: missing module")
		return 1
	}

	module := args[1]
	modPath := module

	// If it's not a file path, search
	if _, err := os.Stat(module); err != nil {
		release, _ := os.ReadFile("/proc/sys/kernel/osrelease")
		kver := strings.TrimSpace(string(release))
		modDir := filepath.Join("/lib/modules", kver)

		filepath.Walk(modDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(path, ".ko") {
				base := filepath.Base(path)
				name := strings.TrimSuffix(base, ".ko")
				if name == module {
					modPath = path
				}
			}
			return nil
		})
	}

	data, err := os.ReadFile(modPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: modinfo: %s: %v\n", module, err)
		return 1
	}

	// Parse modinfo from the .ko file
	content := string(data)
	fmt.Printf("filename:       %s\n", modPath)

	// Extract modinfo strings
	for _, line := range strings.Split(content, "\x00") {
		if strings.HasPrefix(line, "description=") {
			fmt.Printf("description:    %s\n", strings.TrimPrefix(line, "description="))
		}
		if strings.HasPrefix(line, "author=") {
			fmt.Printf("author:         %s\n", strings.TrimPrefix(line, "author="))
		}
		if strings.HasPrefix(line, "license=") {
			fmt.Printf("license:        %s\n", strings.TrimPrefix(line, "license="))
		}
		if strings.HasPrefix(line, "depends=") {
			dep := strings.TrimPrefix(line, "depends=")
			if dep == "" {
				dep = "(unknown)"
			}
			fmt.Printf("depends:        %s\n", dep)
		}
		if strings.HasPrefix(line, "alias=") {
			fmt.Printf("alias:          %s\n", strings.TrimPrefix(line, "alias="))
		}
		if strings.HasPrefix(line, "version=") {
			fmt.Printf("version:        %s\n", strings.TrimPrefix(line, "version="))
		}
	}

	// Get file size
	info, _ := os.Stat(modPath)
	if info != nil {
		fmt.Printf("srcversion:     %x\n", info.ModTime().Unix())
	}

	return 0
}

func depmodMain(args []string) int {
	// Determine kernel release
	release, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		// Try uname -r
		cmd := exec.Command("uname", "-r")
		out, err := cmd.Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, "gobox: depmod: cannot determine kernel version")
			return 1
		}
		release = out
	}
	kver := strings.TrimSpace(string(release))
	modDir := filepath.Join("/lib/modules", kver)

	// Scan for .ko files and extract dependencies
	modules := make(map[string][]string)
	filepath.Walk(modDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".ko") {
			return nil
		}
		relPath, _ := filepath.Rel(modDir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		// Extract depends= from modinfo strings
		var deps []string
		for _, s := range strings.Split(string(data), "\x00") {
			if strings.HasPrefix(s, "depends=") {
				depStr := strings.TrimPrefix(s, "depends=")
				if depStr != "" {
					deps = strings.Split(depStr, ",")
				}
				break
			}
		}
		modules[relPath] = deps
		return nil
	})

	// Write modules.dep
	depPath := filepath.Join(modDir, "modules.dep")
	var lines []string
	for _, mod := range sortedKeys(modules) {
		deps := modules[mod]
		line := mod + ":"
		for _, d := range deps {
			line += " " + d
		}
		lines = append(lines, line)
	}
	output := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(depPath, []byte(output), 0644); err != nil {
		// If we can't write, just print to stdout
		os.Stdout.WriteString(output)
	}
	return 0
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
