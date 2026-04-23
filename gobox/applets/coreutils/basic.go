package coreutils

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gobox/applets"
)

func init() {
	applets.Register("true", applets.AppletFunc(trueMain))
	applets.Register("false", applets.AppletFunc(falseMain))
	applets.Register("echo", applets.AppletFunc(echoMain))
	applets.Register("yes", applets.AppletFunc(yesMain))
	applets.Register("clear", applets.AppletFunc(clearMain))
	applets.Register("nologin", applets.AppletFunc(nologinMain))
	applets.Register("printenv", applets.AppletFunc(printenvMain))
	applets.Register("logname", applets.AppletFunc(lognameMain))
	applets.Register("hostid", applets.AppletFunc(hostidMain))
	applets.Register("whoami", applets.AppletFunc(whoamiMain))
	applets.Register("nproc", applets.AppletFunc(nprocMain))
	applets.Register("link", applets.AppletFunc(linkMain))
	applets.Register("unlink", applets.AppletFunc(unlinkMain))
	applets.Register("tty", applets.AppletFunc(ttyMain))
	applets.Register("sync", applets.AppletFunc(syncMain))
}

func trueMain(args []string) int {
	return 0
}

func falseMain(args []string) int {
	return 1
}

func echoMain(args []string) int {
	n := false
	args0 := args[1:]

	if len(args0) > 0 && args0[0] == "-n" {
		n = true
		args0 = args0[1:]
	}

	out := strings.Join(args0, " ")
	if !n {
		out += "\n"
	}
	os.Stdout.WriteString(out)
	return 0
}

func yesMain(args []string) int {
	s := "y"
	if len(args) > 1 {
		s = strings.Join(args[1:], " ")
	}
	for {
		_, err := fmt.Println(s)
		if err != nil {
			break
		}
	}
	return 0
}

func clearMain(args []string) int {
	os.Stdout.WriteString("\033[H\033[J")
	return 0
}

func nologinMain(args []string) int {
	fmt.Println("This account is currently not available.")
	return 1
}

func printenvMain(args []string) int {
	if len(args) > 1 {
		v := os.Getenv(args[1])
		if v == "" {
			return 1
		}
		fmt.Println(v)
		return 0
	}
	for _, e := range os.Environ() {
		fmt.Println(e)
	}
	return 0
}

func lognameMain(args []string) int {
	// Returns LOGNAME or USER
	name := os.Getenv("LOGNAME")
	if name == "" {
		name = os.Getenv("USER")
	}
	if name == "" {
		fmt.Fprintln(os.Stderr, "gobox: logname: no login name")
		return 1
	}
	fmt.Println(name)
	return 0
}

func hostidMain(args []string) int {
	// Try multiple sources for hostid
	var id string
	for _, path := range []string{"/proc/sys/kernel/hostid", "/etc/hostid"} {
		data, err := os.ReadFile(path)
		if err == nil {
			// data is typically 4 bytes (32-bit) in network byte order
			if len(data) >= 4 {
				id = fmt.Sprintf("%02x%02x%02x%02x", data[0], data[1], data[2], data[3])
			} else {
				id = strings.TrimSpace(string(data))
			}
			break
		}
	}
	if id == "" {
		id = "00000000"
	}
	fmt.Println(id)
	return 0
}

func whoamiMain(args []string) int {
	uid := os.Getuid()
	// Simple: just use LOGNAME/USER like busybox does
	name := os.Getenv("LOGNAME")
	if name == "" {
		name = os.Getenv("USER")
	}
	if name == "" || uid != os.Geteuid() {
		// Fallback: read /proc/self/status or use os/user
		fmt.Fprintln(os.Stderr, "gobox: whoami: cannot find username")
		return 1
	}
	fmt.Println(name)
	return 0
}

func nprocMain(args []string) int {
	// Read number of CPUs from /proc
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: nproc: cannot read /proc/stat")
		return 1
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "cpu") && len(line) > 3 && line[3] >= '0' && line[3] <= '9' {
			count++
		}
	}
	if count == 0 {
		count = 1
	}
	fmt.Println(count)
	return 0
}

func linkMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: link: missing operand")
		return 1
	}
	if err := os.Link(args[1], args[2]); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: link: %v\n", err)
		return 1
	}
	return 0
}

func unlinkMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: unlink: missing operand")
		return 1
	}
	if err := os.Remove(args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: unlink: %v\n", err)
		return 1
	}
	return 0
}

func ttyMain(args []string) int {
	silent := false
	rest := args[1:]
	if len(rest) > 0 && rest[0] == "-s" {
		silent = true
		rest = rest[1:]
	}

	ttyPath, err := os.Readlink("/proc/self/fd/0")
	if err != nil {
		if !silent {
			fmt.Fprintln(os.Stderr, "gobox: tty: not a tty")
		}
		return 1
	}
	if !strings.HasPrefix(ttyPath, "/dev/") {
		if !silent {
			fmt.Fprintln(os.Stderr, "gobox: tty: not a tty")
		}
		return 1
	}
	if !silent {
		fmt.Println(ttyPath)
	}
	return 0
}

func syncMain(args []string) int {
	// syscall.Sync() is the proper way
	syscallSync()
	return 0
}

// io helpers
func readStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}
