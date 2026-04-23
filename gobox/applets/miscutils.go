package applets

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() {
	Register("halt", AppletFunc(haltMain))
	Register("reboot", AppletFunc(rebootMain))
	Register("poweroff", AppletFunc(poweroffMain))
	Register("shutdown", AppletFunc(shutdownMain))
	Register("init", AppletFunc(initMain))
	Register("more", AppletFunc(moreMain))
	Register("less", AppletFunc(lessMain))
	Register("watch", AppletFunc(watchMain))
	Register("logger", AppletFunc(loggerMain))
	Register("reset", AppletFunc(resetMain))
	Register("mktemp", AppletFunc(mktempMain))
	Register("hexdump", AppletFunc(hexdumpMain))
	Register("xxd", AppletFunc(xxdMain))
	Register("uuidgen", AppletFunc(uuidgenMain))
	Register("time", AppletFunc(timeMain))
	Register("cal", AppletFunc(calMain))
	Register("install", AppletFunc(installMain))
	Register("envdir", AppletFunc(envdirMain))
	Register("setsid", AppletFunc(setsidMain))
	Register("flock", AppletFunc(flockMain))
	Register("rev", AppletFunc(revMain))
	Register("id", AppletFunc(idMain))
	Register("groups", AppletFunc(groupsMain))
	Register("last", AppletFunc(lastMain))
	Register("whois", AppletFunc(whoisMain))
	Register("w", AppletFunc(wMain))
	Register("wall", AppletFunc(wallMain))
	Register("lsof", AppletFunc(lsofMain))
	Register("tree", AppletFunc(treeMain))
	Register("bc", AppletFunc(bcMain))
	Register("dc", AppletFunc(dcMain))
	Register("man", AppletFunc(manMain))
}

func haltMain(args []string) int {
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	return 0
}

func rebootMain(args []string) int {
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	return 0
}

func poweroffMain(args []string) int {
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	return 0
}

func shutdownMain(args []string) int {
	now := false
	for _, arg := range args[1:] {
		if arg == "-h" || arg == "-P" || arg == "-r" {
			continue
		}
		if arg == "now" || arg == "+0" {
			now = true
		}
	}
	if now {
		syscall.Sync()
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	}
	fmt.Println("Shutdown scheduled, use 'shutdown now' to power off immediately")
	return 0
}

func initMain(args []string) int {
	// init is PID 1 - in a container, just sleep
	if os.Getpid() == 1 {
		// Simple init: reap children
		for {
			var wstatus syscall.WaitStatus
			pid, err := syscall.Wait4(-1, &wstatus, 0, nil)
			if err != nil {
				time.Sleep(time.Second)
			}
			_ = pid
		}
	}
	fmt.Fprintln(os.Stderr, "gobox: init: not PID 1")
	return 1
}

func moreMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	for _, path := range paths {
		var reader io.Reader
		if path == "" {
			reader = os.Stdin
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: more: %s: %v\n", path, err)
				return 1
			}
			defer f.Close()
			reader = f
		}

		scanner := bufio.NewScanner(reader)
		lines := 0
		for scanner.Scan() {
			fmt.Println(scanner.Text())
			lines++
			if lines >= 24 {
				fmt.Print("--More--")
				bufio.NewScanner(os.Stdin).Scan()
				lines = 0
			}
		}
	}
	return 0
}

func lessMain(args []string) int {
	// less is more with more features, but we implement it as more
	return moreMain(args)
}

func watchMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: watch: missing command")
		return 1
	}

	interval := 2
	cmdArgs := args[1:]

	if args[1] == "-n" && len(args) > 2 {
		interval, _ = strconv.Atoi(args[2])
		if interval < 1 {
			interval = 1
		}
		cmdArgs = args[3:]
	}

	if len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "gobox: watch: missing command")
		return 1
	}

	for {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func loggerMain(args []string) int {
	msg := ""
	if len(args) > 1 {
		msg = strings.Join(args[1:], " ")
	} else {
		data, _ := io.ReadAll(os.Stdin)
		msg = strings.TrimSpace(string(data))
	}

	if msg == "" {
		return 0
	}

	f, err := os.OpenFile("/dev/log", os.O_WRONLY, 0)
	if err != nil {
		// Fall back to /dev/kmsg
		f, err = os.OpenFile("/dev/kmsg", os.O_WRONLY, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, msg)
			return 0
		}
	}
	defer f.Close()
	fmt.Fprintf(f, "<13>%s\n", msg)
	return 0
}

func resetMain(args []string) int {
	os.Stdout.WriteString("\033c")
	return 0
}

func mktempMain(args []string) int {
	template := "tmp.XXXXXX"
	dir := false

	for _, arg := range args[1:] {
		if arg == "-d" {
			dir = true
		} else if !strings.HasPrefix(arg, "-") {
			template = arg
		}
	}

	if strings.Count(template, "X") == 0 {
		template += ".XXXXXX"
	}

	// Replace Xs with random chars
	rand := time.Now().UnixNano()
	name := strings.ReplaceAll(template, "XXXXXX",
		fmt.Sprintf("%06x", rand&0xFFFFFF))

	if dir {
		if err := os.MkdirAll(name, 0700); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: mktemp: %v\n", err)
			return 1
		}
	} else {
		f, err := os.Create(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: mktemp: %v\n", err)
			return 1
		}
		f.Close()
	}

	fmt.Println(name)
	return 0
}

func hexdumpMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	for _, path := range paths {
		var data []byte
		var err error
		if path == "" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: hexdump: %s: %v\n", path, err)
			return 1
		}

		for i := 0; i < len(data); i += 16 {
			fmt.Printf("%08x  ", i)
			for j := 0; j < 8; j++ {
				if i+j < len(data) {
					fmt.Printf("%02x ", data[i+j])
				} else {
					fmt.Print("   ")
				}
			}
			fmt.Print(" ")
			for j := 8; j < 16; j++ {
				if i+j < len(data) {
					fmt.Printf("%02x ", data[i+j])
				} else {
					fmt.Print("   ")
				}
			}
			fmt.Print(" |")
			for j := 0; j < 16 && i+j < len(data); j++ {
				c := data[i+j]
				if c >= 32 && c <= 126 {
					fmt.Printf("%c", c)
				} else {
					fmt.Print(".")
				}
			}
			fmt.Println("|")
		}
		fmt.Printf("%08x\n", len(data))
	}
	return 0
}

func xxdMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	for _, path := range paths {
		var data []byte
		var err error
		if path == "" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: xxd: %s: %v\n", path, err)
			return 1
		}

		for i := 0; i < len(data); i += 16 {
			fmt.Printf("%08x: ", i)
			for j := 0; j < 16; j++ {
				if i+j < len(data) {
					fmt.Printf("%02x ", data[i+j])
				}
			}
			fmt.Print(" ")
			for j := 0; j < 16 && i+j < len(data); j++ {
				c := data[i+j]
				if c >= 32 && c <= 126 {
					fmt.Printf("%c", c)
				} else {
					fmt.Print(".")
				}
			}
			fmt.Println()
		}
	}
	return 0
}

func uuidgenMain(args []string) int {
	// Generate a random UUID v4
	f, err := os.OpenFile("/dev/urandom", os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: uuidgen: cannot read /dev/urandom")
		return 1
	}
	defer f.Close()

	buf := make([]byte, 16)
	io.ReadFull(f, buf)

	// Set version 4
	buf[6] = (buf[6] & 0x0f) | 0x40
	// Set variant bits
	buf[8] = (buf[8] & 0x3f) | 0x80

	fmt.Printf("%08x-%04x-%04x-%04x-%012x\n",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
	return 0
}

func timeMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: time: missing command")
		return 1
	}

	cmd := exec.Command(args[1], args[2:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	fmt.Fprintf(os.Stderr, "real\t%.3fs\n", elapsed.Seconds())

	if err != nil {
		return 1
	}
	return 0
}

func calMain(args []string) int {
	now := time.Now()
	year := now.Year()
	month := now.Month()

	if len(args) > 1 {
		if m, err := strconv.Atoi(args[1]); err == nil && m >= 1 && m <= 12 {
			month = time.Month(m)
		}
	}
	if len(args) > 2 {
		if y, err := strconv.Atoi(args[2]); err == nil {
			year = y
		}
	}

	printCalendar(year, int(month))
	return 0
}

func printCalendar(year, month int) {
	monthNames := []string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}

	fmt.Printf("      %s %d\n", monthNames[month], year)
	fmt.Println("Su Mo Tu We Th Fr Sa")

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local)

	// Print leading spaces
	weekday := firstDay.Weekday()
	for i := 0; i < int(weekday); i++ {
		fmt.Print("   ")
	}

	for day := 1; day <= lastDay.Day(); day++ {
		fmt.Printf("%2d ", day)
		if (int(weekday)+day)%7 == 0 {
			fmt.Println()
		}
	}
	fmt.Println()
}

func installMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: install: missing operand")
		return 1
	}
	// Simple install: copy files with permissions
	mode := os.FileMode(0755)
	dest := args[len(args)-1]
	sources := args[1 : len(args)-1]

	for _, src := range sources {
		data, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: install: %s: %v\n", src, err)
			return 1
		}
		target := dest
		if info, err := os.Stat(dest); err == nil && info.IsDir() {
			target = filepath.Join(dest, filepath.Base(src))
		}
		if err := os.WriteFile(target, data, mode); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: install: %v\n", err)
			return 1
		}
	}
	return 0
}

func envdirMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: envdir: missing operand")
		return 1
	}

	dir := args[1]
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: envdir: %v\n", err)
		return 1
	}

	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		os.Setenv(entry.Name(), strings.TrimSpace(string(data)))
	}

	cmd := exec.Command(args[2], args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return runCommandExit(cmd)
}

func setsidMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: setsid: missing operand")
		return 1
	}
	cmd := exec.Command(args[1], args[2:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return runCommandExit(cmd)
}

func flockMain(args []string) int {
	shared := false
	unlock := false
	nonblock := false
	cmdIndex := -1

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-s", "--shared":
			shared = true
		case "-u", "--unlock":
			unlock = true
		case "-n", "--nb", "--nonblock":
			nonblock = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				cmdIndex = i
				break
			}
			fmt.Fprintf(os.Stderr, "gobox: flock: unknown option: %s\n", args[i])
			return 1
		}
		if cmdIndex > 0 {
			break
		}
	}

	if cmdIndex < 1 || cmdIndex >= len(args) {
		fmt.Fprintln(os.Stderr, "gobox: flock: missing file descriptor or file")
		return 1
	}

	fdPath := args[cmdIndex]
	cmdArgs := args[cmdIndex+1:]

	f, err := os.Open(fdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: flock: %s: %v\n", fdPath, err)
		return 1
	}
	defer f.Close()

	operation := syscall.LOCK_EX
	if shared {
		operation = syscall.LOCK_SH
	}
	if unlock {
		operation = syscall.LOCK_UN
	}
	if nonblock {
		operation |= syscall.LOCK_NB
	}

	if err := syscall.Flock(int(f.Fd()), operation); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: flock: %v\n", err)
		return 1
	}

	if len(cmdArgs) == 0 {
		return 0
	}

	// Run command while holding the lock
	return runAppCommand(cmdArgs[0], cmdArgs[1:])
}

func revMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	for _, path := range paths {
		var scanner *bufio.Scanner
		if path == "" {
			scanner = bufio.NewScanner(os.Stdin)
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: rev: %s: %v\n", path, err)
				return 1
			}
			defer f.Close()
			scanner = bufio.NewScanner(f)
		}
		for scanner.Scan() {
			line := scanner.Text()
			runes := []rune(line)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			fmt.Println(string(runes))
		}
	}
	return 0
}

func runCommandExit(cmd *exec.Cmd) int {
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		return 1
	}
	return 0
}

// Dummy to satisfy sort import (used by netstat)
var _ = sort.Strings

// idMain - print user identity
func idMain(args []string) int {
	uid := os.Getuid()
	gid := os.Getgid()
	username := lookupUserName(uid)
	groupName := lookupGroupName(gid)

	fmt.Printf("uid=%d(%s) gid=%d(%s)", uid, username, gid, groupName)

	// Get supplementary groups
	data, err := os.ReadFile("/proc/self/status")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "Groups:") {
				groups := strings.Fields(line[7:])
				if len(groups) > 0 {
					fmt.Print(" groups=")
					for i, g := range groups {
						gidNum, _ := strconv.Atoi(g)
						gName := lookupGroupName(gidNum)
						if i > 0 {
							fmt.Print(",")
						}
						fmt.Printf("%d(%s)", gidNum, gName)
					}
				}
			}
		}
	}
	fmt.Println()
	return 0
}

// groupsMain - print group memberships
func groupsMain(args []string) int {
	return execTool("groups", args[1:])
}

// lastMain - show last logged in users
func lastMain(args []string) int {
	data, err := os.ReadFile("/var/log/wtmp")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: last: no wtmp file")
		return 1
	}

	type utmp struct {
		Type    int16
		_       [2]byte
		User    [32]byte
		ID      [4]byte
		Line    [32]byte
		Host    [256]byte
		_       [16]byte
		Session int32
		Sec     int32
		Usec    int32
		IP      [4]int32
		_       [20]byte
	}

	entrySize := 384
	for i := 0; i+entrySize <= len(data); i += entrySize {
		var u utmp
		// Manually decode
		user := strings.TrimRight(string(u.User[:]), "\x00")
		line := strings.TrimRight(string(u.Line[:]), "\x00")
		host := strings.TrimRight(string(u.Host[:]), "\x00")
		_ = user
		_ = line
		_ = host
	}

	fmt.Fprintln(os.Stderr, "gobox: last: wtmp parsing limited")
	return 1
}

// whoisMain - whois client
func whoisMain(args []string) int {
	return execTool("whois", args[1:])
}

// wMain - show who is logged on
func wMain(args []string) int {
	return execTool("w", args[1:])
}

// wallMain - send message to all users
func wallMain(args []string) int {
	msg := ""
	if len(args) > 1 {
		msg = strings.Join(args[1:], " ")
	} else {
		data, _ := io.ReadAll(os.Stdin)
		msg = strings.TrimSpace(string(data))
	}

	if msg == "" {
		fmt.Fprintln(os.Stderr, "gobox: wall: no message")
		return 1
	}

	// Write to all terminals
	for _, tty := range []string{"/dev/pts"} {
		entries, err := os.ReadDir(tty)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			path := tty + "/" + entry.Name()
			f, err := os.OpenFile(path, os.O_WRONLY, 0)
			if err != nil {
				continue
			}
			host, _ := os.Hostname()
			fmt.Fprintf(f, "\rBroadcast message from %s@%s (pts/%s):\r\n", os.Getenv("USER"), host, entry.Name())
			fmt.Fprintf(f, "\r%s\r\n", msg)
			f.Close()
		}
	}
	return 0
}

// lsofMain - list open files
func lsofMain(args []string) int {
	return execTool("lsof", args[1:])
}

// treeMain - list directory tree
func treeMain(args []string) int {
	root := "."
	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		root = args[1]
	}

	var printTree func(dir string, prefix string) int
	printTree = func(dir string, prefix string) int {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return 0
		}
		count := 0
		for i, entry := range entries {
			isLast := i == len(entries)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}
			fmt.Printf("%s%s%s\n", prefix, connector, entry.Name())
			count++
			if entry.IsDir() {
				subPrefix := prefix + "│   "
				if isLast {
					subPrefix = prefix + "    "
				}
				count += printTree(filepath.Join(dir, entry.Name()), subPrefix)
			}
		}
		return count
	}

	fmt.Println(root)
	total := printTree(root, "")
	fmt.Printf("\n%d directories, %d files\n", 0, total)
	return 0
}

// bcMain - arbitrary precision calculator
func bcMain(args []string) int {
	return execTool("bc", args[1:])
}

// dcMain - stack-based calculator
func dcMain(args []string) int {
	return execTool("dc", args[1:])
}

// manMain - manual pager
func manMain(args []string) int {
	return execTool("man", args[1:])
}

func lookupUserName(uid int) string {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return fmt.Sprintf("uid=%d", uid)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 {
			uidStr := fields[2]
			if u, err := strconv.Atoi(uidStr); err == nil && u == uid {
				return fields[0]
			}
		}
	}
	return fmt.Sprintf("%d", uid)
}

func lookupGroupName(gid int) string {
	data, err := os.ReadFile("/etc/group")
	if err != nil {
		return fmt.Sprintf("gid=%d", gid)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 {
			gidStr := fields[2]
			if g, err := strconv.Atoi(gidStr); err == nil && g == gid {
				return fields[0]
			}
		}
	}
	return fmt.Sprintf("%d", gid)
}
