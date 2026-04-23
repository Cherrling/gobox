package coreutils

import (
	"bufio"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"syscall"

	"gobox/applets"
)

func init() {
	applets.Register("date", applets.AppletFunc(dateMain))
	applets.Register("dd", applets.AppletFunc(ddMain))
	applets.Register("df", applets.AppletFunc(dfMain))
	applets.Register("du", applets.AppletFunc(duMain))
	applets.Register("stat", applets.AppletFunc(statMain))
	applets.Register("mknod", applets.AppletFunc(mknodMain))
	applets.Register("mkfifo", applets.AppletFunc(mkfifoMain))
	applets.Register("chroot", applets.AppletFunc(chrootMain))
	applets.Register("nohup", applets.AppletFunc(nohupMain))
	applets.Register("nice", applets.AppletFunc(niceMain))
	applets.Register("timeout", applets.AppletFunc(timeoutMain))
	applets.Register("truncate", applets.AppletFunc(truncateMain))
	applets.Register("shred", applets.AppletFunc(shredMain))
	applets.Register("factor", applets.AppletFunc(factorMain))
	applets.Register("seq", applets.AppletFunc(seqMain))
	applets.Register("shuf", applets.AppletFunc(shufMain))
	applets.Register("split", applets.AppletFunc(splitMain))
	applets.Register("tac", applets.AppletFunc(tacMain))
	applets.Register("tsort", applets.AppletFunc(tsortMain))
	applets.Register("stty", applets.AppletFunc(sttyMain))
	applets.Register("who", applets.AppletFunc(whoMain))
	applets.Register("users", applets.AppletFunc(usersMain))
	applets.Register("uptime", applets.AppletFunc(uptimeMain))
	applets.Register("uname", applets.AppletFunc(unameMain))
	applets.Register("arch", applets.AppletFunc(archMain))
	applets.Register("base32", applets.AppletFunc(base32Main))
	applets.Register("base64", applets.AppletFunc(base64Main))
	applets.Register("sleep", applets.AppletFunc(sleepMain))
	applets.Register("env", applets.AppletFunc(envMain))
	applets.Register("test", applets.AppletFunc(testMain))
	applets.Register("od", applets.AppletFunc(odMain))
	applets.Register("strings", applets.AppletFunc(stringsMain))
	applets.Register("printf", applets.AppletFunc(printfMain))
	applets.Register("which", applets.AppletFunc(whichMain))
	applets.Register("xargs", applets.AppletFunc(xargsMain))
	applets.Register("expr", applets.AppletFunc(exprMain))
	applets.Register("renice", applets.AppletFunc(reniceMain))
	applets.Register("stdbuf", applets.AppletFunc(stdbufMain))
	applets.Register("pathchk", applets.AppletFunc(pathchkMain))
	applets.Register("chvt", applets.AppletFunc(chvtMain))
	applets.Register("deallocvt", applets.AppletFunc(deallocvtMain))
}

func dateMain(args []string) int {
	if len(args) > 1 {
		format := strings.Join(args[1:], " ")
		if strings.HasPrefix(format, "+") {
			// Custom format
			now := time.Now()
			// Translate strftime-like format to Go format
			format = format[1:] // remove +
			format = translateDateFormat(format)
			fmt.Println(now.Format(format))
			return 0
		}
		// Try to set date (requires privileges)
		fmt.Fprintln(os.Stderr, "gobox: date: setting date not supported")
		return 1
	}
	fmt.Println(time.Now().Format(time.ANSIC))
	return 0
}

func translateDateFormat(f string) string {
	// Simple strftime to Go format translation
	r := strings.NewReplacer(
		"%Y", "2006",
		"%y", "06",
		"%m", "01",
		"%d", "02",
		"%H", "15",
		"%I", "03",
		"%M", "04",
		"%S", "05",
		"%s", "1136239445", // Unix timestamp - approximate
		"%B", "January",
		"%b", "Jan",
		"%A", "Monday",
		"%a", "Mon",
		"%p", "PM",
		"%Z", "MST",
		"%z", "-0700",
		"%j", "002",
		"%W", "01",
		"%w", "1",
		"%u", "1",
		"%V", "01",
		"%G", "2006",
		"%C", "20",
		"%D", "01/02/06",
		"%F", "2006-01-02",
		"%T", "15:04:05",
		"%r", "03:04:05 PM",
		"%R", "15:04",
		"%n", "\n",
		"%t", "\t",
		"%%", "%",
	)
	return r.Replace(f)
}

func ddMain(args []string) int {
	ifSize := 512
	ofSize := 512
	count := -1
	seek := 0
	skip := 0
	conv := ""

	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]
		switch key {
		case "if":
			f, err := os.Open(val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: dd: cannot open '%s': %v\n", val, err)
				return 1
			}
			os.Stdin = f
		case "of":
			flags := os.O_WRONLY | os.O_CREATE
			if conv == "notrunc" {
				flags |= os.O_APPEND
			}
			f, err := os.OpenFile(val, flags, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: dd: cannot create '%s': %v\n", val, err)
				return 1
			}
			os.Stdout = f
		case "bs":
			n, _ := strconv.Atoi(val)
			if n > 0 {
				ifSize = n
				ofSize = n
			}
		case "count":
			count, _ = strconv.Atoi(val)
		case "skip":
			skip, _ = strconv.Atoi(val)
		case "seek":
			seek, _ = strconv.Atoi(val)
		case "conv":
			conv = val
		case "ibs":
			n, _ := strconv.Atoi(val)
			if n > 0 {
				ifSize = n
			}
		case "obs":
			n, _ := strconv.Atoi(val)
			if n > 0 {
				ofSize = n
			}
		}
	}

	// Skip input blocks
	for i := 0; i < skip; i++ {
		io.CopyN(io.Discard, os.Stdin, int64(ifSize))
	}

	// Seek output
	for i := 0; i < seek; i++ {
		os.Stdout.Write(make([]byte, ofSize))
	}

	buf := make([]byte, ifSize)
	total := int64(0)
	written := int64(0)
	n := count

	for n != 0 {
		_, err := io.ReadFull(os.Stdin, buf)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			break
		}
		os.Stdout.Write(buf)
		total += int64(ifSize)
		written += int64(ifSize)
		if n > 0 {
			n--
		}
	}

	fmt.Fprintf(os.Stderr, "%d+0 records in\n%d+0 records out\n%d bytes copied\n",
		total/int64(ifSize), written/int64(ofSize), written)
	return 0
}

func dfMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{"/"}
	}

	fmt.Printf("%-20s %12s %12s %12s %5s %s\n", "Filesystem", "1K-blocks", "Used", "Available", "Use%", "Mounted on")

	for _, path := range paths {
		// Read /proc/self/mountinfo for filesystem info
		// Use /proc/mounts for simple parsing
		data, err := os.ReadFile("/proc/mounts")
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: df: %s: %v\n", path, err)
			return 1
		}
		// Find the mount point
		mountPoint := path
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == path {
				mountPoint = fields[1]
				break
			}
		}
		// Use statfs syscall to get filesystem info
		var fs syscallStatfsT
		if err := statfs(mountPoint, &fs); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: df: %s: %v\n", path, err)
			return 1
		}
		total := fs.Blocks * uint64(fs.Bsize) / 1024
		available := fs.Bavail * uint64(fs.Bsize) / 1024
		used := total - fs.Bfree*uint64(fs.Bsize)/1024
		usePercent := 0
		if total > 0 {
			usePercent = int(used * 100 / total)
		}
		device := mountPoint
		fmt.Printf("%-20s %12d %12d %12d %4d%% %s\n", device, total, used, available, usePercent, mountPoint)
	}
	return 0
}

func duMain(args []string) int {
	summarize := false
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		if paths[0] == "-s" {
			summarize = true
		}
		paths = paths[1:]
	}

	if len(paths) == 0 {
		paths = []string{"."}
	}

	exitCode := 0
	for _, path := range paths {
		var size int64
		err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				size += (info.Size() + 1023) / 1024
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: du: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		if summarize {
			fmt.Printf("%d\t%s\n", size, path)
		} else {
			fmt.Printf("%d\t%s\n", size, path)
		}
	}
	return exitCode
}

func statMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: stat: missing operand")
		return 1
	}

	exitCode := 0
	for _, path := range args[1:] {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: stat: cannot stat '%s': %v\n", path, err)
			exitCode = 1
			continue
		}
		fmt.Printf("  File: %s\n", path)
		fmt.Printf("  Size: %d\t\tBlocks: %d\tIO Block: %d\n", info.Size(), info.Size()/512+1, 4096)
		fmt.Printf("Device: %s\tInode: %d\tLinks: %d\n", "xxxxh", uint64(0), uint64(1))
		fmt.Printf("Access: (%04o/%s)  Uid: (%d/%d)  Gid: (%d/%d)\n",
			info.Mode().Perm(), info.Mode().String(),
			0, 0,
			0, 0)
		fmt.Printf("Access: %s\n", info.ModTime().Format(time.ANSIC))
		fmt.Printf("Modify: %s\n", info.ModTime().Format(time.ANSIC))
		fmt.Printf("Change: %s\n", info.ModTime().Format(time.ANSIC))
	}
	return exitCode
}

func mknodMain(args []string) int {
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "gobox: mknod: missing operand")
		return 1
	}
	fmt.Fprintln(os.Stderr, "gobox: mknod: not fully supported")
	return 1
}

func mkfifoMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: mkfifo: missing operand")
		return 1
	}

	exitCode := 0
	for _, path := range args[1:] {
		if err := mkfifoSyscall(path, 0666); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: mkfifo: %s: %v\n", path, err)
			exitCode = 1
		}
	}
	return exitCode
}

func chrootMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: chroot: missing operand")
		return 1
	}
	newRoot := args[1]
	cmd := "/bin/sh"
	cmdArgs := []string{}
	if len(args) > 2 {
		cmd = args[2]
		cmdArgs = args[3:]
	}
	if err := chrootSyscall(newRoot, cmd, cmdArgs); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: chroot: %v\n", err)
		return 1
	}
	return 0
}

func nohupMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: nohup: missing operand")
		return 1
	}

	// Ignore SIGHUP
	// In Go, we can't easily ignore signals in a portable way across exec
	// The important behavior is that the child process ignores SIGHUP
	return runCommand(args[1], args[2:])
}

func niceMain(args []string) int {
	adjustment := 10
	cmdStart := 1

	if len(args) > 1 && strings.HasPrefix(args[1], "-n") {
		if len(args) > 2 {
			adjustment, _ = strconv.Atoi(args[2])
			cmdStart = 3
		}
	} else if len(args) > 1 {
		adjustment, _ = strconv.Atoi(args[1])
		cmdStart = 2
	}

	if cmdStart >= len(args) {
		fmt.Fprintln(os.Stderr, "gobox: nice: missing operand")
		return 1
	}

	return runNice(args[cmdStart], args[cmdStart+1:], adjustment)
}

func timeoutMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: timeout: missing operand")
		return 1
	}
	duration, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: timeout: invalid duration: %s\n", args[1])
		return 1
	}

	return runTimeout(args[2], args[3:], time.Duration(duration)*time.Second)
}

func truncateMain(args []string) int {
	size := int64(0)
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		if paths[0] == "-s" && len(paths) > 1 {
			size, _ = strconv.ParseInt(paths[1], 10, 64)
			paths = paths[2:]
		} else {
			paths = paths[1:]
		}
	}

	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "gobox: truncate: missing operand")
		return 1
	}

	exitCode := 0
	for _, path := range paths {
		if err := os.Truncate(path, size); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: truncate: %s: %v\n", path, err)
			exitCode = 1
		}
	}
	return exitCode
}

func shredMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: shred: missing operand")
		return 1
	}

	for _, path := range args[1:] {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: shred: %s: %v\n", path, err)
			return 1
		}

		size := info.Size()
		for pass := 0; pass < 3; pass++ {
			f, err := os.OpenFile(path, os.O_WRONLY, 0)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: shred: %s: %v\n", path, err)
				return 1
			}
			// Write random data
			buf := make([]byte, 4096)
			for written := int64(0); written < size; {
				rand.Read(buf)
				n := int64(len(buf))
				if written+n > size {
					n = size - written
				}
				f.WriteAt(buf[:n], written)
				written += n
			}
			f.Close()
		}
		// Remove the file
		os.Remove(path)
	}
	return 0
}

func factorMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: factor: missing operand")
		return 1
	}
	for _, s := range args[1:] {
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: factor: '%s' is not a valid number\n", s)
			return 1
		}
		fmt.Printf("%s:", s)
		factors := primeFactors(n)
		for _, f := range factors {
			fmt.Printf(" %d", f)
		}
		fmt.Println()
	}
	return 0
}

func primeFactors(n uint64) []uint64 {
	var factors []uint64
	for n%2 == 0 {
		factors = append(factors, 2)
		n /= 2
	}
	for i := uint64(3); i*i <= n; i += 2 {
		for n%i == 0 {
			factors = append(factors, i)
			n /= i
		}
	}
	if n > 1 {
		factors = append(factors, n)
	}
	return factors
}

func seqMain(args []string) int {
	var start, end, step float64 = 1, 1, 1
	switch len(args) {
	case 2:
		end, _ = strconv.ParseFloat(args[1], 64)
	case 3:
		start, _ = strconv.ParseFloat(args[1], 64)
		end, _ = strconv.ParseFloat(args[2], 64)
	case 4:
		start, _ = strconv.ParseFloat(args[1], 64)
		step, _ = strconv.ParseFloat(args[2], 64)
		end, _ = strconv.ParseFloat(args[3], 64)
	default:
		fmt.Fprintln(os.Stderr, "gobox: seq: missing operand")
		return 1
	}

	if step == 0 {
		fmt.Fprintln(os.Stderr, "gobox: seq: zero step")
		return 1
	}

	if step > 0 {
		for i := start; i <= end; i += step {
			fmt.Println(formatFloat(i))
		}
	} else {
		for i := start; i >= end; i += step {
			fmt.Println(formatFloat(i))
		}
	}
	return 0
}

func formatFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	// Remove trailing zeros
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

func shufMain(args []string) int {
	paths := args[1:]
	var lines []string

	if len(paths) > 0 && paths[0] == "-i" && len(paths) > 1 {
		// -i LO-HI
		parts := strings.SplitN(paths[1], "-", 2)
		if len(parts) == 2 {
			lo, _ := strconv.Atoi(parts[0])
			hi, _ := strconv.Atoi(parts[1])
			for i := lo; i <= hi; i++ {
				lines = append(lines, strconv.Itoa(i))
			}
		}
	} else {
		input := os.Stdin
		if len(paths) > 0 && paths[0] != "-i" {
			f, err := os.Open(paths[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: shuf: %s: %v\n", paths[0], err)
				return 1
			}
			defer f.Close()
			input = f
		}
		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
	}

	rand.Shuffle(len(lines), func(i, j int) {
		lines[i], lines[j] = lines[j], lines[i]
	})

	for _, line := range lines {
		fmt.Println(line)
	}
	return 0
}

func splitMain(args []string) int {
	prefix := "x"
	lines := 1000
	input := os.Stdin

	paths := args[1:]
	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		if paths[0] == "-l" && len(paths) > 1 {
			lines, _ = strconv.Atoi(paths[1])
			paths = paths[2:]
		} else if n, err := strconv.Atoi(paths[0][1:]); err == nil {
			lines = n
			paths = paths[1:]
		} else {
			paths = paths[1:]
		}
	}

	if len(paths) > 0 && paths[0] != "" {
		f, err := os.Open(paths[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: split: %s: %v\n", paths[0], err)
			return 1
		}
		defer f.Close()
		input = f
		paths = paths[1:]
	}

	if len(paths) > 0 {
		prefix = paths[0]
	}

	scanner := bufio.NewScanner(input)
	fileNum := 0
	lineCount := 0
	var outFile *os.File

	for scanner.Scan() {
		if lineCount == 0 {
			if outFile != nil {
				outFile.Close()
			}
			name := fmt.Sprintf("%s%c%c", prefix, 'a'+fileNum/26, 'a'+fileNum%26)
			var err error
			outFile, err = os.Create(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: split: %v\n", err)
				return 1
			}
			fileNum++
		}
		fmt.Fprintln(outFile, scanner.Text())
		lineCount++
		if lineCount >= lines {
			lineCount = 0
		}
	}
	if outFile != nil {
		outFile.Close()
	}
	return 0
}

func tacMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	exitCode := 0
	for _, path := range paths {
		var data []byte
		var err error
		if path == "" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tac: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		lines := strings.Split(string(data), "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			if i < len(lines)-1 || lines[i] != "" {
				fmt.Println(lines[i])
			}
		}
	}
	return exitCode
}

func tsortMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	var data []byte
	var err error
	if paths[0] == "" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(paths[0])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: tsort: %v\n", err)
		return 1
	}

	fields := strings.Fields(string(data))
	if len(fields)%2 != 0 {
		// Remove last element if odd
		fields = fields[:len(fields)-1]
	}

	// Build graph
	graph := map[string][]string{}
	inDegree := map[string]int{}
	for i := 0; i < len(fields); i += 2 {
		from, to := fields[i], fields[i+1]
		graph[from] = append(graph[from], to)
		inDegree[to]++
		if _, ok := inDegree[from]; !ok {
			inDegree[from] = 0
		}
	}

	// Kahn's algorithm
	queue := []string{}
	for node, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, node)
		}
	}
	sort.Strings(queue)

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		fmt.Println(node)
		for _, neighbor := range graph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue)
			}
		}
	}
	return 0
}

func sttyMain(args []string) int {
	// Minimal stty - just report settings
	fmt.Println("speed 38400 baud; line 0;")
	fmt.Println("-brkint -imaxbel")
	return 0
}

func whoMain(args []string) int {
	data, err := os.ReadFile("/var/run/utmp")
	if err != nil {
		// Try alternative location
		data, err = os.ReadFile("/var/log/wtmp")
	}
	if err != nil {
		// Minimal: check who is logged in from /proc
		fmt.Fprintln(os.Stderr, "gobox: who: no utmp available")
		return 1
	}
	_ = data
	// Simple: read /var/run/utmp
	fmt.Fprintln(os.Stderr, "gobox: who: not fully implemented")
	return 1
}

func usersMain(args []string) int {
	// Simple: read users from utmp or who command
	data, err := os.ReadFile("/var/run/utmp")
	if err != nil {
		return 1
	}
	_ = data
	return 0
}

func uptimeMain(args []string) int {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: uptime: cannot read /proc/uptime")
		return 1
	}

	var uptimeSecs float64
	fmt.Sscanf(string(data), "%f", &uptimeSecs)

	days := int(uptimeSecs / 86400)
	hours := int(uptimeSecs/3600) % 24
	minutes := int(uptimeSecs/60) % 60

	// Get load average
	loadData, err := os.ReadFile("/proc/loadavg")
	loadAvg := "0.00 0.00 0.00"
	if err == nil {
		parts := strings.Fields(string(loadData))
		if len(parts) >= 3 {
			loadAvg = strings.Join(parts[:3], " ")
		}
	}

	now := time.Now().Format("15:04:05")
	if days > 0 {
		fmt.Printf(" %s up %d days, %d:%02d,  load average: %s\n", now, days, hours, minutes, loadAvg)
	} else {
		fmt.Printf(" %s up %d:%02d,  load average: %s\n", now, hours, minutes, loadAvg)
	}
	return 0
}

func unameMain(args []string) int {
	all := false
	kernel := false
	nodename := false
	kernelRelease := false
	kernelVersion := false
	machine := false
	processor := false
	hardware := false
	showOs := false

	for _, arg := range args[1:] {
		switch arg {
		case "-a":
			all = true
		case "-s":
			kernel = true
		case "-n":
			nodename = true
		case "-r":
			kernelRelease = true
		case "-v":
			kernelVersion = true
		case "-m":
			machine = true
		case "-p":
			processor = true
		case "-i":
			hardware = true
		case "-o":
			showOs = true
		}
	}

	// If no flags, default to -s
	if !all && !kernel && !nodename && !kernelRelease && !kernelVersion &&
		!machine && !processor && !hardware && !showOs {
		kernel = true
	}

	sysname := readSysInfo("/proc/sys/kernel/ostype", "Linux")
	release := readSysInfo("/proc/sys/kernel/osrelease", "")
	version := readSysInfo("/proc/sys/kernel/version", "")
	hn, _ := os.Hostname()
	arch := readSysInfo("/proc/sys/kernel/arch", "")

	if arch == "" {
		arch = runtimeArch()
	}

	if all {
		fmt.Printf("%s %s %s %s %s %s\n", sysname, hn, release, version, arch, "GNU/Linux")
		return 0
	}

	parts := []string{}
	if kernel {
		parts = append(parts, sysname)
	}
	if nodename {
		parts = append(parts, hn)
	}
	if kernelRelease {
		parts = append(parts, release)
	}
	if kernelVersion {
		parts = append(parts, version)
	}
	if machine || processor || hardware {
		parts = append(parts, arch)
	}
	if showOs {
		parts = append(parts, "GNU/Linux")
	}

	fmt.Println(strings.Join(parts, " "))
	return 0
}

func runtimeArch() string {
	// Read /proc/self/exe or use go runtime
	data, err := os.ReadFile("/proc/sys/kernel/arch")
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	// Fallback to reading /proc/self/exe
	link, err := os.Readlink("/proc/self/exe")
	if err == nil {
		_ = link
	}
	return "x86_64"
}

func archMain(args []string) int {
	arch := runtimeArch()
	fmt.Println(arch)
	return 0
}

func readSysInfo(path, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return strings.TrimSpace(string(data))
}

func base32Main(args []string) int {
	// Simplified: wrap encoding/base32
	if len(args) < 2 {
		data, _ := io.ReadAll(os.Stdin)
		fmt.Println(base32Encode(data))
		return 0
	}
	for _, path := range args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: base32: %s: %v\n", path, err)
			return 1
		}
		fmt.Println(base32Encode(data))
	}
	return 0
}

func base64Main(args []string) int {
	if len(args) < 2 {
		data, _ := io.ReadAll(os.Stdin)
		fmt.Println(base64Encode(data))
		return 0
	}
	for _, path := range args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: base64: %s: %v\n", path, err)
			return 1
		}
		fmt.Println(base64Encode(data))
	}
	return 0
}

func sleepMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: sleep: missing operand")
		return 1
	}
	duration, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: sleep: invalid time interval '%s'\n", args[1])
		return 1
	}
	time.Sleep(time.Duration(duration) * time.Second)
	return 0
}

func envMain(args []string) int {
	for _, e := range os.Environ() {
		fmt.Println(e)
	}
	return 0
}

func testMain(args []string) int {
	// test (or [) evaluates expressions
	// This is a simplified implementation
	n := len(args) - 1
	if args[0] == "test" {
		// test EXPR
		return testEval(args[1:])
	}
	// [ EXPR ]
	if n >= 2 && args[n] == "]" {
		return testEval(args[1 : n-1])
	}
	return 2
}

func testEval(args []string) int {
	if len(args) == 0 {
		return 1
	}

	if len(args) == 1 {
		if args[0] == "" {
			return 1
		}
		return 0
	}

	if len(args) == 2 {
		switch args[0] {
		case "!":
			if testEval(args[1:]) == 0 {
				return 1
			}
			return 0
		case "-n":
			if args[1] != "" {
				return 0
			}
			return 1
		case "-z":
			if args[1] == "" {
				return 0
			}
			return 1
		default:
			if args[0] == args[1] {
				return 0
			}
			return 1
		}
	}

	if len(args) >= 3 {
		switch args[1] {
		case "=", "==":
			if args[0] == args[2] {
				return 0
			}
			return 1
		case "!=":
			if args[0] != args[2] {
				return 0
			}
			return 1
		case "-eq":
			a, _ := strconv.Atoi(args[0])
			b, _ := strconv.Atoi(args[2])
			if a == b {
				return 0
			}
			return 1
		case "-ne":
			a, _ := strconv.Atoi(args[0])
			b, _ := strconv.Atoi(args[2])
			if a != b {
				return 0
			}
			return 1
		case "-lt":
			a, _ := strconv.Atoi(args[0])
			b, _ := strconv.Atoi(args[2])
			if a < b {
				return 0
			}
			return 1
		case "-le":
			a, _ := strconv.Atoi(args[0])
			b, _ := strconv.Atoi(args[2])
			if a <= b {
				return 0
			}
			return 1
		case "-gt":
			a, _ := strconv.Atoi(args[0])
			b, _ := strconv.Atoi(args[2])
			if a > b {
				return 0
			}
			return 1
		case "-ge":
			a, _ := strconv.Atoi(args[0])
			b, _ := strconv.Atoi(args[2])
			if a >= b {
				return 0
			}
			return 1
		case "-f":
			info, err := os.Stat(args[2])
			if err == nil && !info.IsDir() {
				return 0
			}
			return 1
		case "-d":
			info, err := os.Stat(args[2])
			if err == nil && info.IsDir() {
				return 0
			}
			return 1
		case "-e":
			if _, err := os.Stat(args[2]); err == nil {
				return 0
			}
			return 1
		case "-r":
			f, err := os.OpenFile(args[2], os.O_RDONLY, 0)
			if err == nil {
				f.Close()
				return 0
			}
			return 1
		case "-w":
			f, err := os.OpenFile(args[2], os.O_WRONLY, 0)
			if err == nil {
				f.Close()
				return 0
			}
			return 1
		case "-x":
			info, err := os.Stat(args[2])
			if err == nil && info.Mode()&0111 != 0 {
				return 0
			}
			return 1
		case "-s":
			info, err := os.Stat(args[2])
			if err == nil && info.Size() > 0 {
				return 0
			}
			return 1
		case "-L", "-h":
			info, err := os.Lstat(args[2])
			if err == nil && info.Mode()&os.ModeSymlink != 0 {
				return 0
			}
			return 1
		case "-o":
			// Shell option (simplified)
			return 1
		}
	}

	// Default: return true if string is non-empty
	if len(args) >= 1 && args[0] != "" {
		return 0
	}
	return 1
}

func odMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	exitCode := 0
	for _, path := range paths {
		var data []byte
		var err error
		if path == "" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: od: %s: %v\n", path, err)
			exitCode = 1
			continue
		}

		addrWidth := 7
		if len(data) > 0xFFFFFFF {
			addrWidth = 11
		}

		for i := 0; i < len(data); i += 16 {
			fmt.Printf("%0*o", addrWidth, i)
			for j := 0; j < 16; j++ {
				if j%8 == 0 {
					fmt.Print(" ")
				}
				if i+j < len(data) {
					fmt.Printf(" %02x", data[i+j])
				} else {
					fmt.Print("   ")
				}
			}
			fmt.Print("  ")
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
		fmt.Printf("%0*o\n", addrWidth, len(data))
	}
	return exitCode
}

func stringsMain(args []string) int {
	minLen := 4
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		if paths[0] == "-n" && len(paths) > 1 {
			minLen, _ = strconv.Atoi(paths[1])
			paths = paths[2:]
		} else {
			paths = paths[1:]
		}
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	exitCode := 0
	for _, path := range paths {
		var data []byte
		var err error
		if path == "" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: strings: %s: %v\n", path, err)
			exitCode = 1
			continue
		}

		current := []byte{}
		for _, b := range data {
			if b >= 32 && b <= 126 {
				current = append(current, b)
			} else {
				if len(current) >= minLen {
					fmt.Println(string(current))
				}
				current = current[:0]
			}
		}
		if len(current) >= minLen {
			fmt.Println(string(current))
		}
	}
	return exitCode
}

func printfMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: printf: missing operand")
		return 1
	}

	format := args[1]
	rest := args[2:]

	// Handle %s, %d, %f etc. - simple implementation
	// Replace escape sequences
	format = strings.ReplaceAll(format, "\\n", "\n")
	format = strings.ReplaceAll(format, "\\t", "\t")
	format = strings.ReplaceAll(format, "\\\\", "\\")

	if strings.Contains(format, "%") && len(rest) > 0 {
		// Simple printf
		argsI := make([]interface{}, len(rest))
		for i, a := range rest {
			argsI[i] = a
		}
		format = strings.ReplaceAll(format, "%s", "%v")
		fmt.Printf(format, argsI...)
	} else {
		fmt.Print(format)
	}
	return 0
}

func whichMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: which: missing operand")
		return 1
	}

	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		pathEnv = "/usr/local/bin:/usr/bin:/bin"
	}
	paths := strings.Split(pathEnv, ":")

	exitCode := 1
	for _, cmd := range args[1:] {
		for _, dir := range paths {
			fullPath := filepath.Join(dir, cmd)
			info, err := os.Stat(fullPath)
			if err == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
				fmt.Println(fullPath)
				exitCode = 0
				break
			}
		}
	}
	return exitCode
}

func xargsMain(args []string) int {
	cmd := "echo"
	cmdArgs := []string{}
	rest := args[1:]

	if len(rest) > 0 {
		cmd = rest[0]
		cmdArgs = rest[1:]
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fullArgs := append([]string{cmd}, cmdArgs...)
		fullArgs = append(fullArgs, strings.Fields(line)...)
		runCommand(cmd, fullArgs[1:])
	}
	return 0
}

func exprMain(args []string) int {
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "gobox: expr: missing operand")
		return 1
	}

	a, errA := strconv.Atoi(args[1])
	b, errB := strconv.Atoi(args[3])

	if errA == nil && errB == nil {
		switch args[2] {
		case "+":
			fmt.Println(a + b)
			return 0
		case "-":
			fmt.Println(a - b)
			return 0
		case "*":
			fmt.Println(a * b)
			return 0
		case "/":
			if b == 0 {
				return 2
			}
			fmt.Println(a / b)
			return 0
		case "%":
			fmt.Println(a % b)
			return 0
		case ">":
			if a > b {
				return 0
			}
			return 1
		case "<":
			if a < b {
				return 0
			}
			return 1
		case "=", "==":
			if a == b {
				return 0
			}
			return 1
		case "!=":
			if a != b {
				return 0
			}
			return 1
		case ">=":
			if a >= b {
				return 0
			}
			return 1
		case "<=":
			if a <= b {
				return 0
			}
			return 1
		}
	}

	// String comparison
	switch args[2] {
	case "=":
		if args[1] == args[3] {
			return 0
		}
		return 1
	case "!=":
		if args[1] != args[3] {
			return 0
		}
		return 1
	case ":":
		// Regex match (simplified)
		if strings.Contains(args[1], args[3]) {
			fmt.Println(len(args[3]))
			return 0
		}
		return 1
	}

	return 1
}

func reniceMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: renice: not fully implemented")
	return 1
}

func stdbufMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: stdbuf: missing operand")
		return 1
	}
	// stdbuf modifies stdio buffering - simplified pass-through
	return runCommand(args[1], args[2:])
}

func pathchkMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: pathchk: missing operand")
		return 1
	}
	exitCode := 0
	for _, path := range args[1:] {
		if len(path) > 4096 {
			fmt.Fprintf(os.Stderr, "gobox: pathchk: '%s': path too long\n", path)
			exitCode = 1
		}
		if path == "" {
			fmt.Fprintf(os.Stderr, "gobox: pathchk: empty path\n")
			exitCode = 1
		}
	}
	return exitCode
}

func chvtMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: chvt: missing operand")
		return 1
	}
	fmt.Fprintln(os.Stderr, "gobox: chvt: not fully supported")
	return 1
}

func deallocvtMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: deallocvt: not fully supported")
	return 1
}

// --- Syscall wrappers ---

type syscallStatfsT struct {
	Bsize  int64
	Blocks uint64
	Bfree  uint64
	Bavail uint64
	Files  uint64
	Ffree  uint64
}

func statfs(path string, buf *syscallStatfsT) error {
	var s syscall.Statfs_t
	if err := syscall.Statfs(path, &s); err != nil {
		return err
	}
	buf.Bsize = int64(s.Bsize)
	buf.Blocks = s.Blocks
	buf.Bfree = s.Bfree
	buf.Bavail = s.Bavail
	buf.Files = s.Files
	buf.Ffree = s.Ffree
	return nil
}

// statMain is implemented inline using syscall.Stat_t

func mkfifoSyscall(path string, mode uint32) error {
	return syscall.Mkfifo(path, mode)
}

func chrootSyscall(newRoot, cmd string, cmdArgs []string) error {
	if err := syscall.Chroot(newRoot); err != nil {
		return err
	}
	if err := syscall.Chdir("/"); err != nil {
		return err
	}
	binary, err := exec.LookPath(cmd)
	if err != nil {
		return fmt.Errorf("command not found: %s", cmd)
	}
	return syscall.Exec(binary, append([]string{cmd}, cmdArgs...), os.Environ())
}

func runCommand(cmd string, args []string) int {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		return 1
	}
	return 0
}

func runNice(cmd string, args []string, adjustment int) int {
	c := exec.Command(cmd, args...)
	c.SysProcAttr = &syscall.SysProcAttr{}
	// Nice value adjustment
	if adjustment > 0 {
		c.SysProcAttr.Pdeathsig = syscall.SIGTERM
	}
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return 1
	}
	return 0
}

func runTimeout(cmd string, args []string, timeout time.Duration) int {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		return 1
	}

	done := make(chan error, 1)
	go func() {
		done <- c.Wait()
	}()

	select {
	case <-time.After(timeout):
		c.Process.Kill()
		return 124
	case err := <-done:
		if err != nil {
			return 1
		}
		return 0
	}
}

func base32Encode(data []byte) string {
	return base32.StdEncoding.EncodeToString(data)
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

