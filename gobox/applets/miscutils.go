package applets

import (
	"bufio"
	"bytes"
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

type hexdumpFmt struct {
	count  int    // repeat count (0 = repeat until end)
	size   int    // bytes per unit (1, 2, 4, 8)
	fmtStr string // printf-like format string
}

func hexdumpMain(args []string) int {
	paths := args[1:]
	formats := []hexdumpFmt{}
	canonical := true // default -C mode

	for len(paths) > 0 && len(paths[0]) > 0 && paths[0][0] == '-' {
		opt := paths[0]
		if opt == "--" {
			paths = paths[1:]
			break
		}
		if opt == "-C" {
			canonical = true
			paths = paths[1:]
			continue
		}
		if opt == "-e" && len(paths) > 1 {
			canonical = false
			parseHexdumpFmt(paths[1], &formats)
			paths = paths[2:]
			continue
		}
		// Unknown option, stop parsing
		break
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	if canonical || len(formats) == 0 {
		// Default -C output
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
			hexdumpCanonical(data)
		}
		return 0
	}

	// -e format mode
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
		hexdumpFormat(data, formats)
	}
	return 0
}

func parseHexdumpFmt(s string, formats *[]hexdumpFmt) {
	// Parse format string: [count/]size "format" [count/]size "format" ...
	i := 0
	for i < len(s) {
		// Skip whitespace
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}

		count := 1
		size := 1

		// Parse count/size (e.g., "16/1" or "/1")
		if s[i] >= '0' && s[i] <= '9' {
			j := i
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			if j < len(s) && s[j] == '/' {
				count, _ = strconv.Atoi(s[i:j])
				i = j + 1
			}
		}
		if i < len(s) && s[i] == '/' {
			i++ // skip leading /
		}
		// Parse size
		if i < len(s) && s[i] >= '0' && s[i] <= '9' {
			j := i
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			size, _ = strconv.Atoi(s[i:j])
			i = j
		}

		// Skip whitespace before format string
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}

		// Parse format string in quotes
		if i < len(s) && s[i] == '"' {
			i++ // skip opening quote
			fmtStart := i
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' && i+1 < len(s) {
					i += 2
				} else {
					i++
				}
			}
			fmtStr := s[fmtStart:i]
			// Unescape the format string
			fmtStr = unescapeHexdumpFmt(fmtStr)
			if i < len(s) {
				i++ // skip closing quote
			}
			*formats = append(*formats, hexdumpFmt{count: count, size: size, fmtStr: fmtStr})
		}
	}
}

func unescapeHexdumpFmt(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case '0':
				b.WriteByte('\x00')
			default:
				b.WriteByte(s[i+1])
			}
			i++
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func hexdumpCanonical(data []byte) {
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

func hexdumpFormat(data []byte, formats []hexdumpFmt) {
	pos := 0
	var prevLine string
	starred := false

	for pos < len(data) {
		var line strings.Builder
		for _, f := range formats {
			// Check if the format string has any % specifiers (consumes data)
			consumesData := strings.Contains(f.fmtStr, "%")
			for k := 0; k < f.count; k++ {
				if !consumesData {
					line.WriteString(f.fmtStr)
					continue
				}
				if pos >= len(data) {
					line.WriteString(applyHexdumpFmt(f.fmtStr, nil, f.size))
				} else {
					unit := make([]byte, f.size)
					n := 0
					for n < f.size && pos < len(data) {
						unit[n] = data[pos]
						n++
						pos++
					}
					line.WriteString(applyHexdumpFmt(f.fmtStr, unit[:n], f.size))
				}
			}
		}
		lineStr := line.String()

		// Handle * repetition marker
		if lineStr == prevLine {
			if !starred {
				fmt.Println("*")
				starred = true
			}
		} else {
			fmt.Print(lineStr)
			prevLine = lineStr
			starred = false
		}
	}
}

func applyHexdumpFmt(fmtStr string, data []byte, size int) string {
	if len(data) == 0 {
		// No data: output blank fields
		return applyHexdumpFmtNoData(fmtStr)
	}

	var b strings.Builder
	i := 0
	for i < len(fmtStr) {
		if fmtStr[i] == '%' {
			i++
			if i >= len(fmtStr) {
				b.WriteByte('%')
				break
			}
			// Parse width
			width := 0
			for i < len(fmtStr) && fmtStr[i] >= '0' && fmtStr[i] <= '9' {
				width = width*10 + int(fmtStr[i]-'0')
				i++
			}
			if i >= len(fmtStr) {
				b.WriteByte('%')
				break
			}
			// Check for display flag after width
			displayFlag := byte(0)
			if fmtStr[i] == '_' || fmtStr[i] == ' ' || fmtStr[i] == '0' {
				displayFlag = fmtStr[i]
				i++
				// If underscore, check for following char
				if displayFlag == '_' && i < len(fmtStr) && (fmtStr[i] == 'c' || fmtStr[i] == 'u' || fmtStr[i] == 'o' || fmtStr[i] == 'x' || fmtStr[i] == 'p') {
					// format like %_c, %_u, %_o, %_x, %_p
					// All _ formats use display character mode:
					// control chars -> names, printable -> char, high bytes -> hex
					b.WriteString(formatHexdumpDisplay(data, width))
					i++
					continue
				}
			}
			if i >= len(fmtStr) {
				b.WriteByte('%')
				break
			}
			switch fmtStr[i] {
			case 'x':
				b.WriteString(formatHexdumpHex(data, size, width))
			case 'd':
				b.WriteString(formatHexdumpInt(data, size, width))
			case 'u':
				b.WriteString(formatHexdumpUint(data, size, width))
			case 'c':
				b.WriteString(formatHexdumpChar(data[0], width))
			case 's':
				b.WriteString(formatHexdumpString(data))
			case 'o':
				b.WriteString(formatHexdumpOctal(data, size, width))
			case '%':
				b.WriteByte('%')
			}
			i++
		} else {
			b.WriteByte(fmtStr[i])
			i++
		}
	}
	return b.String()
}

func applyHexdumpFmtNoData(fmtStr string) string {
	// Apply format with no data: output blank/space-padded fields
	var b strings.Builder
	i := 0
	for i < len(fmtStr) {
		if fmtStr[i] == '%' {
			i++
			if i >= len(fmtStr) {
				b.WriteByte('%')
				break
			}
			// Parse width
			width := 0
			for i < len(fmtStr) && fmtStr[i] >= '0' && fmtStr[i] <= '9' {
				width = width*10 + int(fmtStr[i]-'0')
				i++
			}
			// Skip display flag
			if i < len(fmtStr) && (fmtStr[i] == '_' || fmtStr[i] == ' ' || fmtStr[i] == '0') {
				i++
				if i < len(fmtStr) && (fmtStr[i] == 'c' || fmtStr[i] == 'u' || fmtStr[i] == 'o' || fmtStr[i] == 'x' || fmtStr[i] == 'p') {
					i++
				}
			}
			if i >= len(fmtStr) {
				break
			}
			switch fmtStr[i] {
			case 'x', 'd', 'u', 'o':
				// Output spaces for missing data
				for j := 0; j < width; j++ {
					b.WriteByte(' ')
				}
			case 'c':
				b.WriteByte(' ')
			case 's':
				// nothing
			case '%':
				b.WriteByte('%')
			}
			i++
		} else {
			b.WriteByte(fmtStr[i])
			i++
		}
	}
	return b.String()
}

var hexdumpCharNames = map[int]string{
	0: "nul", 1: "soh", 2: "stx", 3: "etx", 4: "eot", 5: "enq", 6: "ack", 7: "bel",
	8: "bs", 9: "ht", 10: "lf", 11: "vt", 12: "ff", 13: "cr", 14: "so", 15: "si",
	16: "dle", 17: "dc1", 18: "dc2", 19: "dc3", 20: "dc4", 21: "nak", 22: "syn", 23: "etb",
	24: "can", 25: "em", 26: "sub", 27: "esc", 28: "fs", 29: "gs", 30: "rs", 31: "us",
	127: "del",
}

func formatHexdumpDisplay(data []byte, width int) string {
	if len(data) == 0 {
		return ""
	}
	c := data[0]
	// Control chars (0x00-0x1f, 0x7f): 3-letter names
	if c < 32 {
		if name, ok := hexdumpCharNames[int(c)]; ok {
			return fmt.Sprintf("%*s", width, name)
		}
	}
	if c == 127 {
		return fmt.Sprintf("%*s", width, "del")
	}
	// Printable ASCII (0x20-0x7e): the character
	if c >= 32 && c <= 126 {
		return fmt.Sprintf("%*c", width, rune(c))
	}
	// High bytes (0x80-0xff): hex
	return fmt.Sprintf("%*s", width, fmt.Sprintf("%02x", c))
}

func formatHexdumpChar(c byte, width int) string {
	if c >= 32 && c <= 126 {
		return fmt.Sprintf("%*c", width, rune(c))
	}
	switch c {
	case 0:
		return fmt.Sprintf("%*s", width, "nul")
	case 7:
		return fmt.Sprintf("%*s", width, "bel")
	case 8:
		return fmt.Sprintf("%*s", width, "bs")
	case 9:
		return fmt.Sprintf("%*s", width, "ht")
	case 10:
		return fmt.Sprintf("%*s", width, "lf")
	case 11:
		return fmt.Sprintf("%*s", width, "vt")
	case 12:
		return fmt.Sprintf("%*s", width, "ff")
	case 13:
		return fmt.Sprintf("%*s", width, "cr")
	case 27:
		return fmt.Sprintf("%*s", width, "esc")
	default:
		return fmt.Sprintf("%*s", width, "del")
	}
}

func formatHexdumpUint(data []byte, size int, width int) string {
	var val uint64
	switch size {
	case 1:
		val = uint64(data[0])
	case 2:
		val = uint64(data[0]) | uint64(data[1])<<8
	case 4:
		val = uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24
	case 8:
		val = uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24 |
			uint64(data[4])<<32 | uint64(data[5])<<40 | uint64(data[6])<<48 | uint64(data[7])<<56
	}
	if width > 0 {
		return fmt.Sprintf("%*d", width, val)
	}
	return fmt.Sprintf("%d", val)
}

func formatHexdumpInt(data []byte, size int, width int) string {
	var val int64
	switch size {
	case 1:
		val = int64(int8(data[0]))
	case 2:
		val = int64(int16(uint16(data[0]) | uint16(data[1])<<8))
	case 4:
		val = int64(int32(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24))
	case 8:
		val = int64(data[0]) | int64(data[1])<<8 | int64(data[2])<<16 | int64(data[3])<<24 |
			int64(data[4])<<32 | int64(data[5])<<40 | int64(data[6])<<48 | int64(data[7])<<56
	}
	if width > 0 {
		return fmt.Sprintf("%*d", width, val)
	}
	return fmt.Sprintf("%d", val)
}

func formatHexdumpHex(data []byte, size int, width int) string {
	var val uint64
	switch size {
	case 1:
		val = uint64(data[0])
	case 2:
		val = uint64(data[0]) | uint64(data[1])<<8
	case 4:
		val = uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24
	case 8:
		val = uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24 |
			uint64(data[4])<<32 | uint64(data[5])<<40 | uint64(data[6])<<48 | uint64(data[7])<<56
	}
	if width > 0 {
		return fmt.Sprintf("%0*x", width, val)
	}
	return fmt.Sprintf("%x", val)
}

func formatHexdumpOctal(data []byte, size int, width int) string {
	var val uint64
	switch size {
	case 1:
		val = uint64(data[0])
	case 2:
		val = uint64(data[0]) | uint64(data[1])<<8
	case 4:
		val = uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24
	case 8:
		val = uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24 |
			uint64(data[4])<<32 | uint64(data[5])<<40 | uint64(data[6])<<48 | uint64(data[7])<<56
	}
	if width > 0 {
		return fmt.Sprintf("%*o", width, val)
	}
	return fmt.Sprintf("%o", val)
}

func formatHexdumpPointer(data []byte, size int, width int) string {
	var val uint64
	switch size {
	case 1:
		val = uint64(data[0])
	case 2:
		val = uint64(data[0]) | uint64(data[1])<<8
	case 4:
		val = uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24
	case 8:
		val = uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24 |
			uint64(data[4])<<32 | uint64(data[5])<<40 | uint64(data[6])<<48 | uint64(data[7])<<56
	}
	return fmt.Sprintf("0x%x", val)
}

func formatHexdumpString(data []byte) string {
	// Print as printable string (like %s in printf)
	end := len(data)
	if end > 0 && data[end-1] == 0 {
		end--
	}
	var b strings.Builder
	for i := 0; i < end; i++ {
		if data[i] >= 32 && data[i] <= 126 {
			b.WriteByte(data[i])
		} else {
			b.WriteByte('.')
		}
	}
	return b.String()
}

func xxdMain(args []string) int {
	plain := false
	reverse := false
	paths := args[1:]

	// Parse flags
	for len(paths) > 0 && paths[0][0] == '-' {
		switch paths[0] {
		case "-p":
			plain = true
		case "-r":
			reverse = true
		}
		paths = paths[1:]
	}

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

		if reverse {
			out := xxdReverse(data, plain)
			os.Stdout.Write(out)
		} else if plain {
			xxdPlain(data)
		} else {
			xxdNormal(data)
		}
	}
	return 0
}

func xxdNormal(data []byte) {
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

func xxdPlain(data []byte) {
	// Output 30 bytes (60 hex chars) per line, matching busybox -p mode
	for i := 0; i < len(data); i += 30 {
		for j := i; j < i+30 && j < len(data); j++ {
			fmt.Printf("%02x", data[j])
		}
		fmt.Println()
	}
}

func xxdReverse(data []byte, plain bool) []byte {
	var out []byte
	if plain {
		// -p -r mode: nibble1/nibble2 state machine matching busybox behavior
		// badchar is reset each time a full byte is output (for loop iteration)
		p := 0
		for p < len(data) {
			var val byte
			badchar := 0

			// nibble1
			for p < len(data) {
				c := data[p]
				p++
				// In -p mode, skip whitespace before nibble1
				if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
					continue
				}
				if c >= '0' && c <= '9' {
					val = (c - '0') << 4
					break
				} else if (c|0x20) >= 'a' && (c|0x20) <= 'f' {
					val = ((c|0x20) - ('a' - 10)) << 4
					break
				} else {
					// bad char at nibble1
					if badchar > 0 {
						for p < len(data) && data[p] != '\n' {
							p++
						}
						if p < len(data) {
							p++
						}
						goto nextByte
					}
					badchar++
					// continue nibble1 (skip this char)
				}
			}
			if p >= len(data) {
				break
			}

			// nibble2
			for p < len(data) {
				c := data[p]
				p++
				// In -p mode, skip whitespace before nibble2
				if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
					continue
				}
				if c >= '0' && c <= '9' {
					val |= c - '0'
				} else if (c|0x20) >= 'a' && (c|0x20) <= 'f' {
					val |= (c|0x20) - ('a' - 10)
				} else {
					// bad char at nibble2: skip non-hex until next hex, then goto nibble1
					for p < len(data) {
						next := data[p]
						if (next >= '0' && next <= '9') || ((next|0x20) >= 'a' && (next|0x20) <= 'f') {
							break
						}
						p++
					}
					// goto nibble1 (retry with next char)
					break
				}
				out = append(out, val)
				break
			}
		nextByte:
		}
		return out
	}

	// Normal -r: parse "address: hex hex ...  ascii"
	// Uses nibble1/nibble2 state machine (matching busybox behavior)
	for _, line := range strings.Split(string(data), "\n") {
		p := 0
		// Skip leading whitespace
		for p < len(line) && (line[p] == ' ' || line[p] == '\t') {
			p++
		}
		if p >= len(line) {
			continue
		}
		// Skip address (hex digits) and colon
		for p < len(line) && line[p] != ':' && line[p] != ' ' && line[p] != '\t' {
			p++
		}
		if p < len(line) && line[p] == ':' {
			p++
		}

		// nibble1/nibble2 state machine (no whitespace skipping in non-p mode)
		for p < len(line) {
			var val byte
			badchar := 0

			// nibble1
			for p < len(line) {
				c := line[p]
				p++
				if c >= '0' && c <= '9' {
					val = (c - '0') << 4
					break
				} else if (c|0x20) >= 'a' && (c|0x20) <= 'f' {
					val = ((c|0x20) - ('a' - 10)) << 4
					break
				} else {
					if badchar > 0 {
						goto nextLine
					}
					badchar++
				}
			}
			if p >= len(line) {
				break
			}

			// nibble2
			for p < len(line) {
				c := line[p]
				p++
				if c >= '0' && c <= '9' {
					val |= c - '0'
				} else if (c|0x20) >= 'a' && (c|0x20) <= 'f' {
					val |= (c|0x20) - ('a' - 10)
				} else {
					// bad char at nibble2: skip non-hex until next hex, then goto nibble1
					for p < len(line) {
						next := line[p]
						if (next >= '0' && next <= '9') || ((next|0x20) >= 'a' && (next|0x20) <= 'f') {
							break
						}
						p++
					}
					continue
				}
				out = append(out, val)
				break
			}
		}
	nextLine:
	}
	return out
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
	// Parse -f format option
	format := ""
	cmdArgs := args[1:]
	for len(cmdArgs) > 0 {
		if cmdArgs[0] == "-f" && len(cmdArgs) > 1 {
			format = cmdArgs[1]
			cmdArgs = cmdArgs[2:]
		} else if cmdArgs[0] == "--" {
			cmdArgs = cmdArgs[1:]
			break
		} else if strings.HasPrefix(cmdArgs[0], "-") && cmdArgs[0] != "-" {
			// Unknown option, but still treat remaining as command
			break
		} else {
			break
		}
	}

	if len(cmdArgs) < 1 {
		fmt.Fprintln(os.Stderr, "gobox: time: missing command")
		return 1
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	if format == "" {
		fmt.Fprintf(os.Stderr, "real\t%.3fs\n", elapsed.Seconds())
	} else {
		timeFormatOutput(format, cmdArgs[0], elapsed, err)
	}

	if err != nil {
		return 1
	}
	return 0
}

// timeFormatOutput processes a time -f format string and prints to stderr.
// Returns true if output should not have a trailing newline.
func timeFormatOutput(f string, cmdName string, elapsed time.Duration, cmdErr error) {
	skipNewline := false
	for i := 0; i < len(f); i++ {
		c := f[i]

		if c == '\\' {
			i++
			if i >= len(f) {
				// Trailing backslash: prints ?, \, command name (puts adds newline)
				fmt.Fprintf(os.Stderr, "?\\%s\n", cmdName)
				skipNewline = true
				break
			}
			switch f[i] {
			case '\\':
				fmt.Fprint(os.Stderr, "\\")
			case 'n':
				fmt.Fprint(os.Stderr, "\n")
			case 't':
				fmt.Fprint(os.Stderr, "\t")
			default:
				// Unknown escape: ?\X
				fmt.Fprintf(os.Stderr, "?\\%c", f[i])
			}
			continue
		}

		if c == '%' {
			i++
			if i >= len(f) {
				// Trailing percent: prints ?, no newline
				fmt.Fprint(os.Stderr, "?")
				skipNewline = true
				break
			}
			switch f[i] {
			case '%':
				fmt.Fprint(os.Stderr, "%")
			case 'e':
				fmt.Fprintf(os.Stderr, "%.2f", elapsed.Seconds())
			case 'E':
				// h:mm:ss or m:ss format
				total := int(elapsed.Seconds())
				h := total / 3600
				m := (total % 3600) / 60
				s := total % 60
				if h > 0 {
					fmt.Fprintf(os.Stderr, "%d:%02d:%02d", h, m, s)
				} else {
					fmt.Fprintf(os.Stderr, "%d:%02d", m, s)
				}
			default:
				// Unknown format: ?X
				fmt.Fprintf(os.Stderr, "?%c", f[i])
			}
			continue
		}

		fmt.Fprintf(os.Stderr, "%c", c)
	}
	if !skipNewline {
		fmt.Fprint(os.Stderr, "\n")
	}
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

	fmt.Printf("    %s %d\n", monthNames[month], year)
	fmt.Println("Su Mo Tu We Th Fr Sa")

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local)

	// Print leading spaces
	weekday := firstDay.Weekday()
	for i := 0; i < int(weekday); i++ {
		fmt.Print("   ")
	}

	for day := 1; day <= lastDay.Day(); day++ {
		if day > 1 && (int(weekday)+day)%7 != 1 {
			fmt.Print(" ")
		}
		fmt.Printf("%2d", day)
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
		var data []byte
		var err error
		if path == "" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: rev: %s: %v\n", path, err)
			return 1
		}

		// Process line by line, like busybox: fgets reads including newline,
		// strrev uses strlen (stops at NUL), then fputs output.
		start := 0
		for start < len(data) {
			// Find next newline
			nl := bytes.IndexByte(data[start:], '\n')
			var line []byte
			if nl >= 0 {
				line = data[start : start+nl+1] // include newline
				start = start + nl + 1
			} else {
				line = data[start:]
				start = len(data)
			}

			// Truncate at NUL byte (like strlen behavior in C)
			if nulIdx := bytes.IndexByte(line, 0); nulIdx >= 0 {
				line = line[:nulIdx]
			}

			// Reverse the line content (but not the trailing newline)
			hasNewline := len(line) > 0 && line[len(line)-1] == '\n'
			content := line
			if hasNewline {
				content = line[:len(line)-1]
			}
			runes := []rune(string(content))
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			fmt.Print(string(runes))
			if hasNewline {
				fmt.Print("\n")
			}
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
	roots := []string{"."}
	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		roots = args[1:]
	}

	totalDirs := 0
	totalFiles := 0

	for _, root := range roots {
		fi, err := os.Stat(root)
		if err != nil {
			fmt.Printf("%s [error opening dir]\n", root)
			continue
		}

		fmt.Println(root)

		var printTree func(dir string, prefix string) (int, int)
		printTree = func(dir string, prefix string) (int, int) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				return 0, 0
			}

			type entryInfo struct {
				name      string
				isDir     bool
				isSymlink bool
				target    string
			}
			var display []entryInfo
			for _, entry := range entries {
				if len(entry.Name()) > 0 && entry.Name()[0] == '.' {
					continue
				}
				info, err := os.Lstat(filepath.Join(dir, entry.Name()))
				if err != nil {
					continue
				}
				e := entryInfo{name: entry.Name(), isDir: entry.IsDir()}
				if info.Mode()&os.ModeSymlink != 0 {
					e.isSymlink = true
					e.isDir = false
					target, _ := os.Readlink(filepath.Join(dir, entry.Name()))
					e.target = target
				}
				display = append(display, e)
			}

			dirCount := 0
			fileCount := 0
			for i, e := range display {
				isLast := i == len(display)-1
				connector := "├── "
				if isLast {
					connector = "└── "
				}
				line := prefix + connector + e.name
				if e.isSymlink {
					line += " -> " + e.target
				}
				fmt.Println(line)
				if e.isDir {
					subPrefix := prefix + "│   "
					if isLast {
						subPrefix = prefix + "    "
					}
					subDirs, subFiles := printTree(filepath.Join(dir, e.name), subPrefix)
					dirCount += subDirs + 1
					fileCount += subFiles
				} else {
					fileCount++
				}
			}
			return dirCount, fileCount
		}

		if fi.IsDir() {
			dirs, files := printTree(root, "")
			totalDirs += dirs
			totalFiles += files
		} else {
			totalFiles++
		}

	}

	fmt.Println()
	fmt.Printf("%d directories, %d files\n", totalDirs, totalFiles)
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
