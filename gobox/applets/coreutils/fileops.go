package coreutils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gobox/applets"
)

func init() {
	applets.Register("cat", applets.AppletFunc(catMain))
	applets.Register("cp", applets.AppletFunc(cpMain))
	applets.Register("mv", applets.AppletFunc(mvMain))
	applets.Register("rm", applets.AppletFunc(rmMain))
	applets.Register("mkdir", applets.AppletFunc(mkdirMain))
	applets.Register("rmdir", applets.AppletFunc(rmdirMain))
	applets.Register("touch", applets.AppletFunc(touchMain))
	applets.Register("ln", applets.AppletFunc(lnMain))
	applets.Register("chmod", applets.AppletFunc(chmodMain))
	applets.Register("chown", applets.AppletFunc(chownMain))
	applets.Register("chgrp", applets.AppletFunc(chgrpMain))
	applets.Register("ls", applets.AppletFunc(lsMain))
	applets.Register("dirname", applets.AppletFunc(dirnameMain))
	applets.Register("basename", applets.AppletFunc(basenameMain))
	applets.Register("pwd", applets.AppletFunc(pwdMain))
	applets.Register("realpath", applets.AppletFunc(realpathMain))
	applets.Register("readlink", applets.AppletFunc(readlinkMain))
}

func catMain(args []string) int {
	if len(args) < 2 {
		// Read from stdin
		_, err := io.Copy(os.Stdout, os.Stdin)
		if err != nil {
			return 1
		}
		return 0
	}

	exitCode := 0
	for _, path := range args[1:] {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: cat: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		_, err = io.Copy(os.Stdout, f)
		f.Close()
		if err != nil {
			exitCode = 1
		}
	}
	return exitCode
}

func cpMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: cp: missing operand")
		return 1
	}

	recursive := false
	srcArgs := args[1:]

	// Parse options
	for len(srcArgs) > 0 && strings.HasPrefix(srcArgs[0], "-") {
		opt := srcArgs[0]
		if opt == "-r" || opt == "-R" || opt == "--recursive" {
			recursive = true
		} else if opt == "-a" {
			recursive = true
		} else {
			fmt.Fprintf(os.Stderr, "gobox: cp: unknown option: %s\n", opt)
			return 1
		}
		srcArgs = srcArgs[1:]
	}

	if len(srcArgs) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: cp: missing operand")
		return 1
	}

	dest := srcArgs[len(srcArgs)-1]
	sources := srcArgs[:len(srcArgs)-1]

	destInfo, destErr := os.Stat(dest)
	isDir := destErr == nil && destInfo.IsDir()

	for _, src := range sources {
		srcInfo, err := os.Stat(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: cp: %s: %v\n", src, err)
			return 1
		}

		if srcInfo.IsDir() {
			if !recursive {
				fmt.Fprintf(os.Stderr, "gobox: cp: -r not specified; omitting directory '%s'\n", src)
				continue
			}
			var target string
			if isDir {
				target = filepath.Join(dest, filepath.Base(src))
			} else {
				target = dest
			}
			if err := copyDir(src, target); err != nil {
				fmt.Fprintf(os.Stderr, "gobox: cp: %v\n", err)
				return 1
			}
		} else {
			var target string
			if isDir {
				target = filepath.Join(dest, filepath.Base(src))
			} else if len(sources) > 1 {
				fmt.Fprintf(os.Stderr, "gobox: cp: target '%s' is not a directory\n", dest)
				return 1
			} else {
				target = dest
			}
			if err := copyFile(src, target); err != nil {
				fmt.Fprintf(os.Stderr, "gobox: cp: %v\n", err)
				return 1
			}
		}
	}
	return 0
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("cannot open '%s': %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("cannot create '%s': %w", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	// Preserve permissions
	srcInfo, _ := os.Stat(src)
	if srcInfo != nil {
		os.Chmod(dst, srcInfo.Mode())
	}
	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("cannot create directory '%s': %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("cannot read directory '%s': %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func mvMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: mv: missing operand")
		return 1
	}

	dest := args[len(args)-1]
	sources := args[1 : len(args)-1]

	destInfo, destErr := os.Stat(dest)
	isDir := destErr == nil && destInfo.IsDir()

	for _, src := range sources {
		var target string
		if isDir {
			target = filepath.Join(dest, filepath.Base(src))
		} else if len(sources) > 1 {
			fmt.Fprintf(os.Stderr, "gobox: mv: target '%s' is not a directory\n", dest)
			return 1
		} else {
			target = dest
		}

		if err := os.Rename(src, target); err != nil {
			// Cross-device move: copy then delete
			srcInfo, srcErr := os.Stat(src)
			if srcErr != nil {
				fmt.Fprintf(os.Stderr, "gobox: mv: %v\n", srcErr)
				return 1
			}
			if srcInfo.IsDir() {
				if err := copyDir(src, target); err != nil {
					fmt.Fprintf(os.Stderr, "gobox: mv: %v\n", err)
					return 1
				}
				os.RemoveAll(src)
			} else {
				if err := copyFile(src, target); err != nil {
					fmt.Fprintf(os.Stderr, "gobox: mv: %v\n", err)
					return 1
				}
				os.Remove(src)
			}
		}
	}
	return 0
}

func rmMain(args []string) int {
	recursive := false
	force := false
	targets := args[1:]

	for len(targets) > 0 && strings.HasPrefix(targets[0], "-") {
		opt := targets[0]
		if opt == "--" {
			targets = targets[1:]
			break
		}
		for _, c := range opt[1:] {
			switch c {
			case 'r', 'R':
				recursive = true
			case 'f':
				force = true
			default:
				fmt.Fprintf(os.Stderr, "gobox: rm: unknown option: -%c\n", c)
				return 1
			}
		}
		targets = targets[1:]
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "gobox: rm: missing operand")
		return 1
	}

	exitCode := 0
	for _, path := range targets {
		info, err := os.Stat(path)
		if err != nil {
			if force {
				continue
			}
			fmt.Fprintf(os.Stderr, "gobox: rm: cannot remove '%s': %v\n", path, err)
			exitCode = 1
			continue
		}
		if info.IsDir() && !recursive {
			fmt.Fprintf(os.Stderr, "gobox: rm: cannot remove '%s': Is a directory\n", path)
			exitCode = 1
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: rm: cannot remove '%s': %v\n", path, err)
			exitCode = 1
		}
	}
	return exitCode
}

func mkdirMain(args []string) int {
	parents := false
	mode := os.FileMode(0777)
	targets := args[1:]

	for len(targets) > 0 && strings.HasPrefix(targets[0], "-") {
		opt := targets[0]
		if opt == "--" {
			targets = targets[1:]
			break
		}
		for _, c := range opt[1:] {
			switch c {
			case 'p':
				parents = true
			case 'm':
				// mode follows as separate arg
				if len(targets) > 1 {
					m, err := parseMode(targets[1])
					if err != nil {
						fmt.Fprintf(os.Stderr, "gobox: mkdir: invalid mode '%s'\n", targets[1])
						return 1
					}
					mode = m
					targets = targets[1:]
				}
			default:
				fmt.Fprintf(os.Stderr, "gobox: mkdir: unknown option: -%c\n", c)
				return 1
			}
		}
		targets = targets[1:]
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "gobox: mkdir: missing operand")
		return 1
	}

	exitCode := 0
	for _, path := range targets {
		var err error
		if parents {
			err = os.MkdirAll(path, mode)
		} else {
			err = os.Mkdir(path, mode)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: mkdir: cannot create directory '%s': %v\n", path, err)
			exitCode = 1
		}
	}
	return exitCode
}

func rmdirMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: rmdir: missing operand")
		return 1
	}

	exitCode := 0
	for _, path := range args[1:] {
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: rmdir: failed to remove '%s': %v\n", path, err)
			exitCode = 1
		}
	}
	return exitCode
}

func touchMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: touch: missing operand")
		return 1
	}

	exitCode := 0
	for _, path := range args[1:] {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: touch: cannot touch '%s': %v\n", path, err)
			exitCode = 1
			continue
		}
		f.Close()
	}
	return exitCode
}

func lnMain(args []string) int {
	symbolic := false
	force := false
	targets := args[1:]

	for len(targets) > 0 && strings.HasPrefix(targets[0], "-") {
		opt := targets[0]
		if opt == "--" {
			targets = targets[1:]
			break
		}
		for _, c := range opt[1:] {
			switch c {
			case 's':
				symbolic = true
			case 'f':
				force = true
			default:
				fmt.Fprintf(os.Stderr, "gobox: ln: unknown option: -%c\n", c)
				return 1
			}
		}
		targets = targets[1:]
	}

	if len(targets) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: ln: missing operand")
		return 1
	}

	dest := targets[len(targets)-1]
	sources := targets[:len(targets)-1]

	destInfo, destErr := os.Stat(dest)
	isDir := destErr == nil && destInfo.IsDir()

	for _, src := range sources {
		var linkPath string
		if isDir {
			linkPath = filepath.Join(dest, filepath.Base(src))
		} else if len(sources) > 1 {
			fmt.Fprintf(os.Stderr, "gobox: ln: target '%s' is not a directory\n", dest)
			return 1
		} else {
			linkPath = dest
		}

		if force {
			os.Remove(linkPath)
		}

		var err error
		if symbolic {
			err = os.Symlink(src, linkPath)
		} else {
			err = os.Link(src, linkPath)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: ln: failed to create link: %v\n", err)
			return 1
		}
	}
	return 0
}

func chmodMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: chmod: missing operand")
		return 1
	}

	mode, err := parseMode(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: chmod: invalid mode: %s\n", args[1])
		return 1
	}

	exitCode := 0
	for _, path := range args[2:] {
		if err := os.Chmod(path, mode); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: chmod: %s: %v\n", path, err)
			exitCode = 1
		}
	}
	return exitCode
}

func chownMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: chown: missing operand")
		return 1
	}

	// Parse owner[:group]
	ownerGroup := args[1]
	var uid, gid int = -1, -1

	if parts := strings.SplitN(ownerGroup, ":", 2); len(parts) >= 1 && parts[0] != "" {
		uid = findUID(parts[0])
		if uid < 0 {
			fmt.Fprintf(os.Stderr, "gobox: chown: unknown user: %s\n", parts[0])
			return 1
		}
		if len(parts) > 1 && parts[1] != "" {
			gid = findGID(parts[1])
			if gid < 0 {
				fmt.Fprintf(os.Stderr, "gobox: chown: unknown group: %s\n", parts[1])
				return 1
			}
		}
	}

	exitCode := 0
	for _, path := range args[2:] {
		if err := lchown(path, uid, gid); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: chown: %s: %v\n", path, err)
			exitCode = 1
		}
	}
	return exitCode
}

func chgrpMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: chgrp: missing operand")
		return 1
	}

	gid := findGID(args[1])
	if gid < 0 {
		fmt.Fprintf(os.Stderr, "gobox: chgrp: unknown group: %s\n", args[1])
		return 1
	}

	exitCode := 0
	for _, path := range args[2:] {
		if err := lchown(path, -1, gid); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: chgrp: %s: %v\n", path, err)
			exitCode = 1
		}
	}
	return exitCode
}

func lsMain(args []string) int {
	targets := args[1:]
	if len(targets) == 0 {
		targets = []string{"."}
	}

	exitCode := 0
	for _, path := range targets {
		entries, err := os.ReadDir(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: ls: cannot access '%s': %v\n", path, err)
			exitCode = 1
			continue
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		if len(targets) > 1 {
			fmt.Printf("%s:\n", path)
		}
		for _, name := range names {
			fmt.Println(name)
		}
		if len(targets) > 1 {
			fmt.Println()
		}
	}
	return exitCode
}

func dirnameMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: dirname: missing operand")
		return 1
	}
	for _, path := range args[1:] {
		fmt.Println(filepath.Dir(path))
	}
	return 0
}

func basenameMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: basename: missing operand")
		return 1
	}
	path := args[1]
	suffix := ""
	if len(args) > 2 {
		suffix = args[2]
	}
	base := filepath.Base(path)
	if suffix != "" && strings.HasSuffix(base, suffix) {
		base = base[:len(base)-len(suffix)]
	}
	fmt.Println(base)
	return 0
}

func pwdMain(args []string) int {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: pwd: cannot get current directory")
		return 1
	}
	fmt.Println(wd)
	return 0
}

func realpathMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: realpath: missing operand")
		return 1
	}
	path, err := filepath.EvalSymlinks(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: realpath: %s: %v\n", args[1], err)
		return 1
	}
	fmt.Println(path)
	return 0
}

func readlinkMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: readlink: missing operand")
		return 1
	}
	path, err := os.Readlink(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: readlink: %s: %v\n", args[1], err)
		return 1
	}
	fmt.Println(path)
	return 0
}

// parseMode parses a numeric file mode string (e.g., "755", "0644").
func parseMode(s string) (os.FileMode, error) {
	var mode uint32
	if _, err := fmt.Sscanf(s, "%o", &mode); err != nil {
		return 0, fmt.Errorf("invalid mode: %s", s)
	}
	return os.FileMode(mode), nil
}

// findUID looks up a username and returns its UID.
// Simple implementation using /etc/passwd.
func findUID(name string) int {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return -1
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[0] == name {
			var uid int
			fmt.Sscanf(parts[2], "%d", &uid)
			return uid
		}
	}
	return -1
}

// findGID looks up a group name and returns its GID.
func findGID(name string) int {
	data, err := os.ReadFile("/etc/group")
	if err != nil {
		return -1
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[0] == name {
			var gid int
			fmt.Sscanf(parts[2], "%d", &gid)
			return gid
		}
	}
	return -1
}
