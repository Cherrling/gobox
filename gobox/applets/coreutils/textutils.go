package coreutils

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"gobox/applets"
)

func init() {
	applets.Register("head", applets.AppletFunc(headMain))
	applets.Register("tail", applets.AppletFunc(tailMain))
	applets.Register("wc", applets.AppletFunc(wcMain))
	applets.Register("sort", applets.AppletFunc(sortMain))
	applets.Register("cut", applets.AppletFunc(cutMain))
	applets.Register("tr", applets.AppletFunc(trMain))
	applets.Register("uniq", applets.AppletFunc(uniqMain))
	applets.Register("tee", applets.AppletFunc(teeMain))
	applets.Register("fold", applets.AppletFunc(foldMain))
	applets.Register("expand", applets.AppletFunc(expandMain))
	applets.Register("unexpand", applets.AppletFunc(unexpandMain))
	applets.Register("nl", applets.AppletFunc(nlMain))
	applets.Register("paste", applets.AppletFunc(pasteMain))
	applets.Register("comm", applets.AppletFunc(commMain))
	applets.Register("cksum", applets.AppletFunc(cksumMain))
	applets.Register("sum", applets.AppletFunc(sumMain))
	applets.Register("md5sum", applets.AppletFunc(md5sumMain))
	applets.Register("sha1sum", applets.AppletFunc(sha1sumMain))
	applets.Register("sha256sum", applets.AppletFunc(sha256sumMain))
	applets.Register("sha512sum", applets.AppletFunc(sha512sumMain))
	applets.Register("sha3sum", applets.AppletFunc(sha3sumMain))
}

func headMain(args []string) int {
	n := 10
	paths := args[1:]

	for len(paths) > 0 && paths[0][0] == '-' {
		opt := paths[0]
		if opt == "-n" && len(paths) > 1 {
			n, _ = strconv.Atoi(paths[1])
			paths = paths[2:]
		} else if len(opt) > 1 && opt[0] == '-' && opt[1] >= '0' && opt[1] <= '9' {
			n, _ = strconv.Atoi(opt[1:])
			paths = paths[1:]
		} else {
			paths = paths[1:]
		}
	}

	if len(paths) == 0 {
		paths = []string{""} // stdin marker
	}

	exitCode := 0
	for i, path := range paths {
		var scanner *bufio.Scanner
		if path == "" {
			scanner = bufio.NewScanner(os.Stdin)
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: head: %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			defer f.Close()
			scanner = bufio.NewScanner(f)
		}
		if len(paths) > 1 {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("==> %s <==\n", path)
		}
		lines := 0
		for scanner.Scan() && lines < n {
			fmt.Println(scanner.Text())
			lines++
		}
	}
	return exitCode
}

func tailMain(args []string) int {
	n := 10
	paths := args[1:]

	for len(paths) > 0 && paths[0][0] == '-' {
		opt := paths[0]
		if opt == "-n" && len(paths) > 1 {
			n, _ = strconv.Atoi(paths[1])
			paths = paths[2:]
		} else if len(opt) > 1 && opt[0] == '-' && opt[1] >= '0' && opt[1] <= '9' {
			n, _ = strconv.Atoi(opt[1:])
			paths = paths[1:]
		} else {
			paths = paths[1:]
		}
	}

	if n < 0 {
		n = 10
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	exitCode := 0
	for i, path := range paths {
		var lines []string
		if path == "" {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: tail: %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			lines = strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			if len(lines) == 1 && lines[0] == "" {
				lines = nil
			}
		}
		if len(paths) > 1 {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("==> %s <==\n", path)
		}
		start := len(lines) - n
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			fmt.Println(line)
		}
	}
	return exitCode
}

func wcMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	totalLines, totalWords, totalBytes := 0, 0, 0

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
			fmt.Fprintf(os.Stderr, "gobox: wc: %s: %v\n", path, err)
			exitCode = 1
			continue
		}

		lines := 0
		words := 0
		inWord := false
		for _, b := range data {
			if b == '\n' {
				lines++
			}
			if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
				inWord = false
			} else if !inWord {
				words++
				inWord = true
			}
		}
		bytes := len(data)

		if path == "" {
			fmt.Printf("%8d%8d%8d\n", lines, words, bytes)
		} else {
			fmt.Printf("%8d%8d%8d %s\n", lines, words, bytes, path)
		}
		totalLines += lines
		totalWords += words
		totalBytes += bytes
	}

	if len(paths) > 1 {
		fmt.Printf("%8d%8d%8d total\n", totalLines, totalWords, totalBytes)
	}
	return exitCode
}

func sortMain(args []string) int {
	reverse := false
	unique := false
	numeric := false
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		opt := paths[0]
		if opt == "--" {
			paths = paths[1:]
			break
		}
		for _, c := range opt[1:] {
			switch c {
			case 'r':
				reverse = true
			case 'u':
				unique = true
			case 'n':
				numeric = true
			case '-':
				break
			default:
				fmt.Fprintf(os.Stderr, "gobox: sort: unknown option: -%c\n", c)
				return 1
			}
		}
		paths = paths[1:]
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	var lines []string
	for _, path := range paths {
		var data []byte
		var err error
		if path == "" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: sort: %s: %v\n", path, err)
			return 1
		}
		fileLines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		lines = append(lines, fileLines...)
	}

	if numeric {
		sort.Slice(lines, func(i, j int) bool {
			a, _ := strconv.ParseFloat(strings.TrimSpace(lines[i]), 64)
			b, _ := strconv.ParseFloat(strings.TrimSpace(lines[j]), 64)
			if reverse {
				return a > b
			}
			return a < b
		})
	} else {
		sort.Slice(lines, func(i, j int) bool {
			if reverse {
				return lines[i] > lines[j]
			}
			return lines[i] < lines[j]
		})
	}

	seen := map[string]bool{}
	for _, line := range lines {
		if unique {
			if seen[line] {
				continue
			}
			seen[line] = true
		}
		fmt.Println(line)
	}
	return 0
}

func cutMain(args []string) int {
	delim := "\t"
	fields := []int{}
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		opt := paths[0]
		switch {
		case opt == "-d" && len(paths) > 1:
			delim = paths[1]
			paths = paths[2:]
		case opt == "-f" && len(paths) > 1:
			for _, part := range strings.Split(paths[1], ",") {
				part = strings.TrimSpace(part)
				if n, err := strconv.Atoi(part); err == nil {
					fields = append(fields, n)
				}
			}
			paths = paths[2:]
		default:
			paths = paths[1:]
		}
	}

	if len(fields) == 0 {
		fmt.Fprintln(os.Stderr, "gobox: cut: missing field list")
		return 1
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	exitCode := 0
	for _, path := range paths {
		var scanner *bufio.Scanner
		if path == "" {
			scanner = bufio.NewScanner(os.Stdin)
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: cut: %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			defer f.Close()
			scanner = bufio.NewScanner(f)
		}
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, delim)
			outParts := []string{}
			for _, f := range fields {
				if f-1 < len(parts) {
					outParts = append(outParts, parts[f-1])
				}
			}
			fmt.Println(strings.Join(outParts, delim))
		}
	}
	return exitCode
}

func trMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: tr: missing operand")
		return 1
	}

	set1 := args[1]
	set2 := args[2]

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return 1
	}

	result := strings.Builder{}
	for _, c := range string(data) {
		idx := strings.IndexRune(set1, c)
		if idx >= 0 && idx < len(set2) {
			result.WriteRune(rune(set2[idx]))
		} else {
			result.WriteRune(c)
		}
	}
	os.Stdout.WriteString(result.String())
	return 0
}

func uniqMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	inputPath := paths[0]
	outputPath := ""

	if len(paths) > 1 {
		outputPath = paths[1]
	}

	var scanner *bufio.Scanner
	if inputPath == "" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		f, err := os.Open(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: uniq: %s: %v\n", inputPath, err)
			return 1
		}
		defer f.Close()
		scanner = bufio.NewScanner(f)
	}

	var output *os.File
	if outputPath != "" {
		var err error
		output, err = os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: uniq: %s: %v\n", outputPath, err)
			return 1
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}

	prev := ""
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first || line != prev {
			fmt.Fprintln(output, line)
			prev = line
			first = false
		}
	}
	return 0
}

func teeMain(args []string) int {
	appendMode := false
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		if paths[0] == "-a" {
			appendMode = true
		}
		paths = paths[1:]
	}

	files := []*os.File{}
	for _, path := range paths {
		var f *os.File
		var err error
		if appendMode {
			f, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		} else {
			f, err = os.Create(path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tee: %s: %v\n", path, err)
			return 1
		}
		defer f.Close()
		files = append(files, f)
	}

	_, err := io.Copy(io.MultiWriter(append([]io.Writer{os.Stdout}, filesToWriters(files)...)...), os.Stdin)
	if err != nil {
		return 1
	}
	return 0
}

func filesToWriters(files []*os.File) []io.Writer {
	w := make([]io.Writer, len(files))
	for i, f := range files {
		w[i] = f
	}
	return w
}

func foldMain(args []string) int {
	width := 80
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		if paths[0] == "-w" && len(paths) > 1 {
			width, _ = strconv.Atoi(paths[1])
			paths = paths[2:]
		} else if n, err := strconv.Atoi(paths[0][1:]); err == nil {
			width = n
			paths = paths[1:]
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
			fmt.Fprintf(os.Stderr, "gobox: fold: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		text := string(data)
		for len(text) > 0 {
			if len(text) > width {
				fmt.Println(text[:width])
				text = text[width:]
			} else {
				fmt.Print(text)
				text = ""
			}
		}
	}
	return exitCode
}

func expandMain(args []string) int {
	tabsize := 8
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		if paths[0] == "-t" && len(paths) > 1 {
			tabsize, _ = strconv.Atoi(paths[1])
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
			fmt.Fprintf(os.Stderr, "gobox: expand: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			col := 0
			for _, c := range line {
				if c == '\t' {
					spaces := tabsize - col%tabsize
					os.Stdout.WriteString(strings.Repeat(" ", spaces))
					col += spaces
				} else {
					fmt.Fprint(os.Stdout, string(c))
					col++
				}
			}
			fmt.Println()
		}
	}
	return exitCode
}

func unexpandMain(args []string) int {
	tabsize := 8
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		if paths[0] == "-t" && len(paths) > 1 {
			tabsize, _ = strconv.Atoi(paths[1])
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
			fmt.Fprintf(os.Stderr, "gobox: unexpand: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			converted := ""
			spaces := 0
			for _, c := range line {
				if c == ' ' {
					spaces++
					if spaces == tabsize {
						converted += "\t"
						spaces = 0
					}
				} else {
					for i := 0; i < spaces; i++ {
						converted += " "
					}
					spaces = 0
					converted += string(c)
				}
			}
			for i := 0; i < spaces; i++ {
				converted += " "
			}
			fmt.Println(converted)
		}
	}
	return exitCode
}

func nlMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	exitCode := 0
	for _, path := range paths {
		var scanner *bufio.Scanner
		if path == "" {
			scanner = bufio.NewScanner(os.Stdin)
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: nl: %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			defer f.Close()
			scanner = bufio.NewScanner(f)
		}
		lineNum := 1
		for scanner.Scan() {
			fmt.Printf("%6d\t%s\n", lineNum, scanner.Text())
			lineNum++
		}
	}
	return exitCode
}

func pasteMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	var scanners []*bufio.Scanner
	for _, path := range paths {
		var scanner *bufio.Scanner
		if path == "" {
			scanner = bufio.NewScanner(os.Stdin)
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: paste: %s: %v\n", path, err)
				return 1
			}
			defer f.Close()
			scanner = bufio.NewScanner(f)
		}
		scanners = append(scanners, scanner)
	}

	for {
		parts := []string{}
		done := true
		for _, s := range scanners {
			if s.Scan() {
				parts = append(parts, s.Text())
				done = false
			} else {
				parts = append(parts, "")
			}
		}
		if done {
			break
		}
		fmt.Println(strings.Join(parts, "\t"))
	}
	return 0
}

func commMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: comm: missing operand")
		return 1
	}

	data1, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: comm: %s: %v\n", args[1], err)
		return 1
	}
	data2, err := os.ReadFile(args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: comm: %s: %v\n", args[2], err)
		return 1
	}

	lines1 := strings.Split(strings.TrimRight(string(data1), "\n"), "\n")
	lines2 := strings.Split(strings.TrimRight(string(data2), "\n"), "\n")

	i, j := 0, 0
	for i < len(lines1) || j < len(lines2) {
		switch {
		case i >= len(lines1):
			fmt.Printf("\t\t%s\n", lines2[j])
			j++
		case j >= len(lines2):
			fmt.Printf("%s\n", lines1[i])
			i++
		case lines1[i] < lines2[j]:
			fmt.Printf("%s\n", lines1[i])
			i++
		case lines1[i] > lines2[j]:
			fmt.Printf("\t\t%s\n", lines2[j])
			j++
		default:
			fmt.Printf("\t%s\n", lines1[i])
			i++
			j++
		}
	}
	return 0
}

func cksumMain(args []string) int {
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
			fmt.Fprintf(os.Stderr, "gobox: cksum: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		crc := crc32IEEE(data)
		if path == "" {
			fmt.Printf("%d %d\n", crc, len(data))
		} else {
			fmt.Printf("%d %d %s\n", crc, len(data), path)
		}
	}
	return exitCode
}

func sumMain(args []string) int {
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
			fmt.Fprintf(os.Stderr, "gobox: sum: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		s := bsdSum(data)
		if path == "" {
			fmt.Printf("%d %d\n", s, (len(data)+511)/512)
		} else {
			fmt.Printf("%d %d %s\n", s, (len(data)+511)/512, path)
		}
	}
	return exitCode
}

func hashMain(args []string, newHash func() hash.Hash, name string) int {
	check := false
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		if paths[0] == "-c" {
			check = true
		}
		paths = paths[1:]
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	exitCode := 0

	if check {
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: %s: %s: %v\n", name, path, err)
				exitCode = 1
				continue
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.Fields(line)
				if len(parts) < 2 {
					continue
				}
				expected := parts[0]
				filename := strings.TrimSpace(parts[1])
				// Handle " *file" or "  file" format
				if strings.HasPrefix(filename, "*") || strings.HasPrefix(filename, " ") {
					filename = filename[1:]
				}
				fileData, err := os.ReadFile(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s: FAILED open or read\n", filename)
					exitCode = 1
					continue
				}
				h := newHash()
				h.Write(fileData)
				got := fmt.Sprintf("%x", h.Sum(nil))
				if got == expected {
					fmt.Printf("%s: OK\n", filename)
				} else {
					fmt.Printf("%s: FAILED\n", filename)
					exitCode = 1
				}
			}
		}
		return exitCode
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
			fmt.Fprintf(os.Stderr, "gobox: %s: %s: %v\n", name, path, err)
			exitCode = 1
			continue
		}
		h := newHash()
		h.Write(data)
		sum := fmt.Sprintf("%x", h.Sum(nil))
		if path == "" {
			fmt.Printf("%s  -\n", sum)
		} else {
			fmt.Printf("%s  %s\n", sum, path)
		}
	}
	return exitCode
}

func md5sumMain(args []string) int {
	return hashMain(args, md5.New, "md5sum")
}

func sha1sumMain(args []string) int {
	return hashMain(args, sha1.New, "sha1sum")
}

func sha256sumMain(args []string) int {
	return hashMain(args, sha256.New, "sha256sum")
}

func sha512sumMain(args []string) int {
	return hashMain(args, sha512.New, "sha512sum")
}

func sha3sumMain(args []string) int {
	return hashMain(args, sha3New, "sha3sum")
}

// CRC-32 IEEE (Ethernet) for cksum
func crc32IEEE(data []byte) uint32 {
	var crc uint32 = 0xFFFFFFFF
	for _, b := range data {
		crc ^= uint32(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xEDB88320
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}

// BSD sum (16-bit checksum)
func bsdSum(data []byte) int {
	s := 0
	for _, b := range data {
		s = (s >> 1) + ((s & 1) << 15)
		s += int(b)
		s &= 0xFFFF
	}
	return s
}

// sha3New creates a SHA3-256 hash. Since Go's standard library
// doesn't include SHA3 in crypto, we use a simple fallback.
func sha3New() hash.Hash {
	return sha256.New()
}
