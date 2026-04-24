package coreutils

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha3"
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
		paths = []string{""}
	}

	exitCode := 0
	for i, path := range paths {
		var reader io.Reader
		if path == "" {
			reader = os.Stdin
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: head: %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			defer f.Close()
			reader = f
		}
		if len(paths) > 1 {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("==> %s <==\n", path)
		}
		if n >= 0 {
			scanner := bufio.NewScanner(reader)
			for j := 0; j < n && scanner.Scan(); j++ {
				fmt.Println(scanner.Text())
			}
		} else {
			var lines []string
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			keep := len(lines) + n
			if keep < 0 {
				keep = 0
			}
			for i := 0; i < keep; i++ {
				fmt.Println(lines[i])
			}
		}
	}
	return exitCode
}

func tailMain(args []string) int {
	n := 10
	countBytes := false
	fromStart := false
	paths := args[1:]

	for len(paths) > 0 && paths[0][0] == '-' {
		opt := paths[0]
		if opt == "--" {
			paths = paths[1:]
			break
		}
		if opt == "-c" && len(paths) > 1 {
			countBytes = true
			arg := paths[1]
			if len(arg) > 0 && arg[0] == '+' {
				fromStart = true
				n, _ = strconv.Atoi(arg[1:])
			} else {
				n, _ = strconv.Atoi(arg)
			}
			paths = paths[2:]
		} else if opt == "-n" && len(paths) > 1 {
			arg := paths[1]
			if len(arg) > 0 && arg[0] == '+' {
				fromStart = true
				n, _ = strconv.Atoi(arg[1:])
			} else {
				n, _ = strconv.Atoi(arg)
			}
			paths = paths[2:]
		} else if len(opt) > 1 && opt[1] >= '0' && opt[1] <= '9' {
			n, _ = strconv.Atoi(opt[1:])
			paths = paths[1:]
		} else {
			paths = paths[1:]
		}
	}

	if n < 1 {
		n = 10
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	exitCode := 0
	for i, path := range paths {
		var data []byte
		var err error
		if path == "" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tail: %s: %v\n", path, err)
			exitCode = 1
			continue
		}

		if len(paths) > 1 {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("==> %s <==\n", path)
		}

		if countBytes {
			if fromStart {
				start := n - 1
				if start >= len(data) {
					start = len(data)
				}
				os.Stdout.Write(data[start:])
			} else {
				start := len(data) - n
				if start < 0 {
					start = 0
				}
				os.Stdout.Write(data[start:])
			}
		} else {
			s := string(data)
			// Strip only one trailing newline to preserve blank lines
			if len(s) > 0 && s[len(s)-1] == '\n' {
				s = s[:len(s)-1]
			}
			lines := strings.Split(s, "\n")
			if fromStart {
				start := n - 1
				if start >= len(lines) {
					start = len(lines)
				}
				for _, line := range lines[start:] {
					fmt.Println(line)
				}
			} else {
				start := len(lines) - n
				if start < 0 {
					start = 0
				}
				for _, line := range lines[start:] {
					fmt.Println(line)
				}
			}
		}
	}
	return exitCode
}

func wcMain(args []string) int {
	countLines := true
	countWords := true
	countBytes := true
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		opt := paths[0]
		if opt == "--" {
			paths = paths[1:]
			break
		}
		countLines = false
		countWords = false
		countBytes = false
		for _, c := range opt[1:] {
			switch c {
			case 'l':
				countLines = true
			case 'w':
				countWords = true
			case 'c':
				countBytes = true
			}
		}
		paths = paths[1:]
	}

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

		count := 0
		if countLines { count++ }
		if countWords { count++ }
		if countBytes { count++ }
		padded := count > 1

		if path == "" {
			if countLines {
				if padded { fmt.Printf("%8d", lines) } else { fmt.Printf("%d", lines) }
			}
			if countWords {
				if padded { fmt.Printf("%8d", words) } else { fmt.Printf("%d", words) }
			}
			if countBytes {
				if padded { fmt.Printf("%8d", bytes) } else { fmt.Printf("%d", bytes) }
			}
			fmt.Println()
		} else {
			if countLines { fmt.Printf("%8d", lines) }
			if countWords { fmt.Printf("%8d", words) }
			if countBytes { fmt.Printf("%8d", bytes) }
			fmt.Printf(" %s\n", path)
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

// parseLeadingFloat extracts a floating-point number from the beginning of s.
// Like C's strtod, it stops at the first non-numeric character.
func parseLeadingFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// Find where the number ends
	end := 0
	if s[0] == '-' || s[0] == '+' {
		end = 1
	}
	hasDot := false
	for end < len(s) {
		c := s[end]
		if c >= '0' && c <= '9' {
			end++
		} else if c == '.' && !hasDot {
			hasDot = true
			end++
		} else {
			break
		}
	}
	if end == 0 || (end == 1 && (s[0] == '-' || s[0] == '+')) {
		return 0
	}
	val, _ := strconv.ParseFloat(s[:end], 64)
	return val
}

func sortMain(args []string) int {
	reverse := false
	unique := false
	numeric := false
	ignoreBlanks := false
	stable := false
	monthSort := false
	humanNumeric := false
	foldCase := false
	nullTerminated := false
	outputFile := ""
	delim := ""
	type keySpec struct {
		startField int // 1-indexed, 0 = beginning of line
		startChar  int // 0 = beginning of field
		endField   int // 0 = end of line
		endChar    int // 0 = end of field
		n          bool
		r          bool
		b          bool
		M          bool
		h          bool
	}
	keys := []keySpec{}

	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		opt := paths[0]
		if opt == "--" {
			paths = paths[1:]
			break
		}
		if opt == "-" {
			break
		}
		if strings.HasPrefix(opt, "-o") {
			if len(opt) > 2 {
				outputFile = opt[2:]
			} else if len(paths) > 1 {
				outputFile = paths[1]
				paths = paths[1:]
			}
			paths = paths[1:]
			continue
		}
		if strings.HasPrefix(opt, "-t") {
			if len(opt) > 2 {
				delim = opt[2:]
			} else if len(paths) > 1 {
				delim = paths[1]
				paths = paths[1:]
			}
			paths = paths[1:]
			continue
		}
		if strings.HasPrefix(opt, "-k") {
			ks := keySpec{n: numeric, r: false, b: ignoreBlanks}
			keyStr := ""
			if len(opt) > 2 {
				keyStr = opt[2:]
			} else if len(paths) > 1 {
				keyStr = paths[1]
				paths = paths[1:]
			}
			// Parse key: pos1[,pos2][flags]
			parts := strings.SplitN(keyStr, ",", 2)
			// Parse pos1
			parseKeyPos := func(s string) (field, char int, flags string) {
				s = strings.TrimSpace(s)
				// Extract trailing flags (letters)
				flagsEnd := len(s)
				for flagsEnd > 0 && strings.ContainsRune("nrbMh", rune(s[flagsEnd-1])) {
					flagsEnd--
				}
				flags = s[flagsEnd:]
				s = s[:flagsEnd]
				// Parse field.char
				if dotIdx := strings.Index(s, "."); dotIdx >= 0 {
					field, _ = strconv.Atoi(s[:dotIdx])
					char, _ = strconv.Atoi(s[dotIdx+1:])
				} else if s != "" {
					field, _ = strconv.Atoi(s)
				}
				return
			}
			startField, startChar, flags1 := parseKeyPos(parts[0])
			ks.startField = startField
			ks.startChar = startChar
			for _, f := range flags1 {
				switch f {
				case 'n':
					ks.n = true
				case 'r':
					ks.r = true
				case 'b':
					ks.b = true
				case 'M':
					ks.M = true
					monthSort = true
				case 'h':
					ks.h = true
					humanNumeric = true
				}
			}
			if len(parts) > 1 {
				endField, endChar, flags2 := parseKeyPos(parts[1])
				ks.endField = endField
				ks.endChar = endChar
				for _, f := range flags2 {
					switch f {
					case 'n':
						ks.n = true
					case 'r':
						ks.r = true
					case 'b':
						ks.b = true
					case 'M':
						ks.M = true
						monthSort = true
					case 'h':
						ks.h = true
						humanNumeric = true
					}
				}
			}
			keys = append(keys, ks)
			paths = paths[1:]
			continue
		}
		for _, c := range opt[1:] {
			switch c {
			case 'r':
				reverse = true
			case 'u':
				unique = true
			case 'n':
				numeric = true
			case 'b':
				ignoreBlanks = true
			case 's':
				stable = true
			case 'h':
				humanNumeric = true
			case 'M':
				monthSort = true
			case 'f':
				foldCase = true
			case 'z':
				nullTerminated = true
			case '-':
				break
			default:
				fmt.Fprintf(os.Stderr, "gobox: sort: unknown option: -%c\n", c)
				return 1
			}
		}
		paths = paths[1:]
	}

	// Apply global -r to keys without type-specific flags (after all options are parsed)
	for i := range keys {
		if reverse && !keys[i].n && !keys[i].M && !keys[i].h {
			keys[i].r = true
		}
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	var rawLines []string
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
		if nullTerminated {
			// Split by NUL, trim trailing NUL
			sdata := string(data)
			sdata = strings.TrimRight(sdata, "\x00")
			if sdata != "" {
				rawLines = append(rawLines, strings.Split(sdata, "\x00")...)
			}
		} else {
			sdata := string(data)
			// Preserve trailing newline behavior
			if len(sdata) > 0 && sdata[len(sdata)-1] == '\n' {
				sdata = sdata[:len(sdata)-1]
			}
			if sdata != "" {
				rawLines = append(rawLines, strings.Split(sdata, "\n")...)
			}
		}
	}

	type line struct {
		text   string
		index  int
	}

	lines := make([]line, len(rawLines))
	for i, l := range rawLines {
		lines[i] = line{text: l, index: i}
	}

	// Month name lookup
	monthNames := map[string]int{
		"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
		"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
	}

	// Human-readable suffix values
		_ = map[string]float64{} // placeholder

	// Extract key from a line for comparison
	extractKey := func(text string, ks keySpec) string {
		// Split into fields
		var fields []string
		if delim == "" {
			fields = strings.Fields(text)
		} else {
			fields = strings.Split(text, delim)
		}

		startIdx := ks.startField - 1
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx >= len(fields) {
			return ""
		}

		endIdx := ks.endField - 1
		if endIdx < 0 || endIdx >= len(fields) {
			endIdx = len(fields) - 1
		}
		if ks.endField == 0 {
			endIdx = len(fields) - 1
		}

		// Build key from fields
		key := ""
		for fi := startIdx; fi <= endIdx; fi++ {
			if fi > startIdx {
				if delim == "" {
					key += " "
				} else {
					key += delim
				}
			}
			f := fields[fi]
			if fi == startIdx && ks.startChar > 0 && ks.startChar <= len(f) {
				f = f[ks.startChar-1:]
			}
			if fi == endIdx && ks.endChar > 0 && ks.endChar <= len(f) {
				f = f[:ks.endChar]
			}
			key += f
		}
		return key
	}

	// Compare two values
	compare := func(a, b string, useNumeric, useMonth, useHuman bool) int {
		if foldCase {
			a = strings.ToLower(a)
			b = strings.ToLower(b)
		}
		if ignoreBlanks {
			a = strings.TrimLeft(a, " \t")
			b = strings.TrimLeft(b, " \t")
		}
		if useMonth {
			aShort := strings.ToLower(a)
			bShort := strings.ToLower(b)
			if len(aShort) > 3 {
				aShort = aShort[:3]
			}
			if len(bShort) > 3 {
				bShort = bShort[:3]
			}
			am := monthNames[aShort]
			bm := monthNames[bShort]
			if am != bm {
				if am < bm {
					return -1
				}
				return 1
			}
			return 0
		}
		if useHuman {
			parseHuman := func(s string) (float64, int) {
				s = strings.TrimSpace(s)
				if s == "" {
					return 0, -1
				}
				val := parseLeadingFloat(s)
				// Find where the number ended
				numLen := 0
				if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
					numLen = 1
				}
				hasDot := false
				for numLen < len(s) {
					c := s[numLen]
					if c >= '0' && c <= '9' {
						numLen++
					} else if c == '.' && !hasDot {
						hasDot = true
						numLen++
					} else {
						break
					}
				}
				tail := ""
				if numLen < len(s) {
					tail = s[numLen:]
				}
				if tail == "" {
					return val, -1
				}
				// Check suffix (busybox logic: only k/K for lowercase, 
				// uppercase M/G/T/P/E/Z/Y, SI units)
				suffixMap := map[string]int{
					"K": 0, "M": 1, "G": 2, "T": 3, "P": 4, "E": 5, "Z": 6, "Y": 7,
					"k": 0,
				}
				firstChar := string(tail[0])
				if idx, ok := suffixMap[firstChar]; ok {
					return val, idx
				}
				return val, -1
			}
			av, ai := parseHuman(a)
			bv, bi := parseHuman(b)
			if ai != bi {
				if ai < bi {
					return -1
				}
				return 1
			}
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
		}
		if useNumeric {
			av := parseLeadingFloat(a)
			bv := parseLeadingFloat(b)
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
		}
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}

	less := func(i, j int) bool {
		li := lines[i]
		lj := lines[j]

		if len(keys) > 0 {
			for _, ks := range keys {
				ki := extractKey(li.text, ks)
				kj := extractKey(lj.text, ks)
				cmp := compare(ki, kj, ks.n, ks.M, ks.h)
				if cmp != 0 {
					if ks.r {
						return cmp > 0
					}
					return cmp < 0
				}
			}
			// Keys equal: use full line as tiebreaker
			if stable {
				return li.index < lj.index
			}
			cmp := compare(li.text, lj.text, numeric, monthSort, humanNumeric)
			if cmp != 0 {
				if reverse {
					return cmp > 0
				}
				return cmp < 0
			}
			return li.text < lj.text
		}

		// No keys: compare full lines
		if stable {
			return li.index < lj.index
		}
		cmp := compare(li.text, lj.text, numeric, monthSort, humanNumeric)
		if cmp != 0 {
			if reverse {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	}

	sort.Slice(lines, less)

	// Handle -u based on key comparison
	if unique {
		type lineGroup struct {
			first line
		}
		var uniq []line
		for _, l := range lines {
			if len(uniq) == 0 {
				uniq = append(uniq, l)
				continue
			}
			last := uniq[len(uniq)-1]
			// Compare last and current
			equal := false
			if len(keys) > 0 {
				equal = true
				for _, ks := range keys {
					ki := extractKey(last.text, ks)
					kj := extractKey(l.text, ks)
					if compare(ki, kj, ks.n, false, false) != 0 {
						equal = false
						break
					}
				}
			} else {
				if compare(last.text, l.text, numeric, monthSort, humanNumeric) == 0 {
					equal = true
				}
			}
			if !equal {
				uniq = append(uniq, l)
			}
		}
		lines = uniq
	}

	// Output
	out := os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: sort: %s: %v\n", outputFile, err)
			return 1
		}
		defer f.Close()
		out = f
	}

	sep := "\n"
	if nullTerminated {
		sep = "\x00"
	}
	for i, l := range lines {
		if i > 0 {
			fmt.Fprint(out, sep)
		}
		fmt.Fprint(out, l.text)
	}
	if len(lines) > 0 {
		fmt.Fprint(out, sep)
	}
	return 0
}

func cutMain(args []string) int {
	delim := "\t"
	outDelim := "\t"
	fields := []int{}
	bytes := false
	chars := false
	suppress := false
	regexMode := false
	paths := args[1:]

	type fieldRange struct {
		start, end int // end == 0 means open-ended
	}
	var ranges []fieldRange

	parseFieldList := func(s string) {
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if strings.Contains(part, "-") {
				parts := strings.SplitN(part, "-", 2)
				start := 0
				if parts[0] != "" {
					start, _ = strconv.Atoi(parts[0])
				}
				end := 0
				if len(parts) > 1 && parts[1] != "" {
					n, err := strconv.Atoi(parts[1])
					if err == nil {
						end = n
					}
				}
				if start > 0 && end > 0 && start > end {
					continue
				}
				ranges = append(ranges, fieldRange{start, end})
			} else {
				if n, err := strconv.Atoi(part); err == nil {
					ranges = append(ranges, fieldRange{n, n})
				}
			}
		}
	}

loop:
	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		opt := paths[0]
		if opt == "--" {
			paths = paths[1:]
			break
		}
		switch {
		case opt == "-d" && len(paths) > 1:
			delim = paths[1]
			outDelim = delim
			paths = paths[2:]
		case strings.HasPrefix(opt, "-d") && len(opt) > 2:
			delim = opt[2:]
			outDelim = delim
			paths = paths[1:]
		case opt == "-D":
			// no sorting (ignored)
			paths = paths[1:]
		case strings.HasPrefix(opt, "-D") && len(opt) > 2:
			// Combined -D with other options, re-insert the rest
			paths = append([]string{"-" + opt[2:]}, paths[1:]...)
		case opt == "-F" && len(paths) > 1:
			regexMode = true
			delim = " "
			outDelim = " "
			parseFieldList(paths[1])
			paths = paths[2:]
		case strings.HasPrefix(opt, "-F"):
			regexMode = true
			delim = " "
			outDelim = " "
			parseFieldList(opt[2:])
			paths = paths[1:]
		case opt == "-f" && len(paths) > 1:
			parseFieldList(paths[1])
			paths = paths[2:]
		case strings.HasPrefix(opt, "-f"):
			parseFieldList(opt[2:])
			paths = paths[1:]
		case opt == "-b" && len(paths) > 1:
			bytes = true
			parseFieldList(paths[1])
			paths = paths[2:]
		case strings.HasPrefix(opt, "-b"):
			bytes = true
			parseFieldList(opt[2:])
			paths = paths[1:]
		case opt == "-c" && len(paths) > 1:
			chars = true
			parseFieldList(paths[1])
			paths = paths[2:]
		case strings.HasPrefix(opt, "-c"):
			chars = true
			parseFieldList(opt[2:])
			paths = paths[1:]
		case opt == "-s":
			suppress = true
			paths = paths[1:]
		case opt == "-n":
			paths = paths[1:]
		case opt == "-":
			paths = paths[1:]
			paths = append([]string{"-"}, paths...)
			break loop
		default:
			paths = paths[1:]
		}
	}

	if len(ranges) == 0 {
		// Check if the next path looks like a positional field list
		if len(paths) > 0 && len(paths[0]) > 0 && paths[0][0] >= '0' && paths[0][0] <= '9' {
			parseFieldList(paths[0])
			paths = paths[1:]
		}
	}
	if len(ranges) == 0 {
		fmt.Fprintln(os.Stderr, "gobox: cut: missing field list")
		return 1
	}

	// Expand ranges into field list
	for _, r := range ranges {
		if r.end > 0 {
			s := r.start
			if s < 1 {
				s = 1
			}
			for n := s; n <= r.end; n++ {
				fields = append(fields, n)
			}
		} else if r.start > 0 {
			// open-ended range N- : mark with sentinel 0 after start
			fields = append(fields, r.start, 0)
		}
	}

	// Deduplicate (preserve order) and find sentinel 0
	seen := map[int]bool{}
	unique := []int{}
	hadZero := false
	for _, f := range fields {
		if f == 0 {
			hadZero = true
			continue
		}
		if seen[f] {
			continue
		}
		seen[f] = true
		unique = append(unique, f)
	}
	if hadZero {
		unique = append(unique, 0)
	}
	fields = unique

	// Find open-ended start position
	hasOpenEnd := false
	openStart := 0
	for i, f := range fields {
		if f == 0 && i > 0 {
			hasOpenEnd = true
			openStart = fields[i-1]
			break
		}
	}

	// Handle stdin via "-"
	if len(paths) == 0 {
		paths = []string{""}
	} else {
		var expanded []string
		for _, p := range paths {
			if p == "-" {
				expanded = append(expanded, "")
			} else {
				expanded = append(expanded, p)
			}
		}
		paths = expanded
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
			if bytes || chars {
				outParts := []string{}
				for _, f := range fields {
					if f < 1 {
						continue
					}
					if f-1 < len(line) {
						outParts = append(outParts, string(line[f-1]))
					}
				}
				if hasOpenEnd && openStart > 0 {
					for i := openStart; i < len(line); i++ {
						outParts = append(outParts, string(line[i]))
					}
				}
				fmt.Println(strings.Join(outParts, ""))
			} else {
				parts := strings.Split(line, delim)
				if suppress && !strings.Contains(line, delim) {
					continue
				}
				if !strings.Contains(line, delim) {
					if regexMode {
						fmt.Println()
					} else {
						fmt.Println(line)
					}
					continue
				}
				outParts := []string{}
				for _, f := range fields {
					if f < 1 {
						continue
					}
					if f-1 < len(parts) {
						outParts = append(outParts, parts[f-1])
					}
				}
				if hasOpenEnd && openStart > 0 {
					for i := openStart; i < len(parts); i++ {
						outParts = append(outParts, parts[i])
					}
				}
				fmt.Println(strings.Join(outParts, outDelim))
			}
		}
	}
	return exitCode
}

func expandTRSet(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		// Check for [:class:] inside brackets
		if s[i] == '[' && i+3 < len(s) && s[i+1] == ':' {
			closeIdx := strings.Index(s[i+2:], ":]")
			if closeIdx >= 0 {
				className := s[i+2 : i+2+closeIdx]
				switch className {
				case "alnum":
					for c := '0'; c <= '9'; c++ { result.WriteRune(c) }
					for c := 'A'; c <= 'Z'; c++ { result.WriteRune(c) }
					for c := 'a'; c <= 'z'; c++ { result.WriteRune(c) }
				case "alpha":
					for c := 'A'; c <= 'Z'; c++ { result.WriteRune(c) }
					for c := 'a'; c <= 'z'; c++ { result.WriteRune(c) }
				case "digit":
					for c := '0'; c <= '9'; c++ { result.WriteRune(c) }
				case "xdigit":
					for c := '0'; c <= '9'; c++ { result.WriteRune(c) }
					for c := 'A'; c <= 'F'; c++ { result.WriteRune(c) }
					for c := 'a'; c <= 'f'; c++ { result.WriteRune(c) }
				case "lower":
					for c := 'a'; c <= 'z'; c++ { result.WriteRune(c) }
				case "upper":
					for c := 'A'; c <= 'Z'; c++ { result.WriteRune(c) }
				case "space":
					result.WriteString(" \t\n\r\f\v")
				case "blank":
					result.WriteString(" \t")
				case "punct":
					for c := '!'; c <= '/'; c++ { result.WriteRune(c) }
					for c := ':'; c <= '@'; c++ { result.WriteRune(c) }
					for c := '['; c <= 96; c++ { result.WriteRune(rune(c)) }
					for c := '{'; c <= '~'; c++ { result.WriteRune(c) }
				case "cntrl":
					for c := 0; c <= 31; c++ { result.WriteRune(rune(c)) }
					result.WriteRune(127)
				case "graph":
					for c := '!'; c <= '~'; c++ { result.WriteRune(c) }
				case "print":
					for c := ' '; c <= '~'; c++ { result.WriteRune(c) }
				}
				i += closeIdx + 4 // skip past [:class:]
				continue
			}
		}
		// Check for range X-Y (but not if X is '[' - that's handled above)
		if i+2 < len(s) && s[i+1] == '-' && s[i] != '[' {
			start := s[i]
			end := s[i+2]
			for c := start; c <= end; c++ {
				result.WriteByte(c)
			}
			i += 3
			continue
		}
		// For '[' not followed by ':', output it literally (brackets are not special for ranges)
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

func trMain(args []string) int {
	complement := false
	deleteMode := false

	i := 1
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			break
		}
		for j := 1; j < len(arg); j++ {
			switch arg[j] {
			case 'c':
				complement = true
			case 'd':
				deleteMode = true
			}
		}
		i++
	}

	remaining := args[i:]

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return 1
	}

	if deleteMode {
		set1 := ""
		if len(remaining) >= 1 {
			set1 = expandTRSet(remaining[0])
		}
		var delSet map[rune]bool
		if complement {
			// Delete everything NOT in set1
			delSet = make(map[rune]bool)
			for _, c := range set1 {
				delSet[c] = true
			}
			var result strings.Builder
			for _, c := range string(data) {
				if delSet[c] {
					result.WriteRune(c)
				}
			}
			os.Stdout.WriteString(result.String())
		} else {
			// Delete everything IN set1
			delSet = make(map[rune]bool)
			for _, c := range set1 {
				delSet[c] = true
			}
			var result strings.Builder
			for _, c := range string(data) {
				if !delSet[c] {
					result.WriteRune(c)
				}
			}
			os.Stdout.WriteString(result.String())
		}
		return 0
	}

	set1 := ""
	set2 := ""
	if len(remaining) >= 1 {
		set1 = expandTRSet(remaining[0])
	}
	if len(remaining) >= 2 {
		set2 = expandTRSet(remaining[1])
	}

	if complement {
		// Build the full character set
		fullSet := make(map[rune]bool)
		for _, c := range set1 {
			fullSet[c] = true
		}
		// set2 should have one more char than set1 for complement
		// (last char of set2 is used for all complemented chars)
		complementChar := byte(0)
		if len(set2) > 0 {
			complementChar = set2[len(set2)-1]
		}

		var result strings.Builder
		for _, c := range string(data) {
			if !fullSet[c] {
				// Char is NOT in set1, map to complementChar
				if complementChar != 0 {
					result.WriteByte(complementChar)
				}
			} else {
				// Char is in set1, try to map
				idx := strings.IndexRune(set1, c)
				if idx >= 0 && idx < len(set2) {
					result.WriteByte(set2[idx])
				} else {
					result.WriteRune(c)
				}
			}
		}
		os.Stdout.WriteString(result.String())
		return 0
	}

	// Simple translation
	var result strings.Builder
	for _, c := range string(data) {
		idx := strings.IndexRune(set1, c)
		if idx >= 0 && idx < len(set2) {
			result.WriteByte(set2[idx])
		} else {
			result.WriteRune(c)
		}
	}
	os.Stdout.WriteString(result.String())
	return 0
}

func uniqMain(args []string) int {
	count := false
	dupsOnly := false
	uniqOnly := false
	skipFields := 0
	skipChars := 0
	maxChars := -1

	i := 1
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			break
		}
		for j := 1; j < len(arg); j++ {
			switch arg[j] {
			case 'c':
				count = true
			case 'd':
				dupsOnly = true
			case 'u':
				uniqOnly = true
			case 'f':
				if j+1 < len(arg) {
					skipFields, _ = strconv.Atoi(arg[j+1:])
					j = len(arg)
				} else if i+1 < len(args) {
					i++
					skipFields, _ = strconv.Atoi(args[i])
				}
			case 's':
				if j+1 < len(arg) {
					skipChars, _ = strconv.Atoi(arg[j+1:])
					j = len(arg)
				} else if i+1 < len(args) {
					i++
					skipChars, _ = strconv.Atoi(args[i])
				}
			case 'w':
				if j+1 < len(arg) {
					maxChars, _ = strconv.Atoi(arg[j+1:])
					j = len(arg)
				} else if i+1 < len(args) {
					i++
					maxChars, _ = strconv.Atoi(args[i])
				}
			}
		}
		i++
	}

	paths := args[i:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	inputPath := paths[0]
	outputPath := ""

	if len(paths) > 1 {
		if paths[1] == "-" {
			outputPath = ""
		} else {
			outputPath = paths[1]
		}
	}

	var scanner *bufio.Scanner
	if inputPath == "" || inputPath == "-" {
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

	compareLine := func(line string) string {
		s := line
		// Skip fields (busybox compatible: skip whitespace, then non-whitespace)
		for f := 0; f < skipFields; f++ {
			s = strings.TrimLeft(s, " \t")
			for len(s) > 0 && s[0] != ' ' && s[0] != '\t' {
				s = s[1:]
			}
		}
		// Skip chars (byte by byte)
		for i := 0; i < skipChars && len(s) > 0; i++ {
			s = s[1:]
		}
		// Apply max chars
		if maxChars >= 0 && maxChars < len(s) {
			s = s[:maxChars]
		}
		return s
	}

	prev := ""
	prevKey := ""
	first := true
	countNum := 1

	for scanner.Scan() {
		line := scanner.Text()
		key := compareLine(line)

		if first {
			prev = line
			prevKey = key
			first = false
			countNum = 1
			continue
		}

		if key == prevKey {
			countNum++
			continue
		}

		// Output previous group
		if dupsOnly && uniqOnly {
			// -d and -u together produce no output
		} else if count && !dupsOnly && !uniqOnly {
			fmt.Fprintf(output, "%d %s\n", countNum, prev)
		} else if dupsOnly && countNum > 1 {
			fmt.Fprintln(output, prev)
		} else if uniqOnly && countNum == 1 {
			fmt.Fprintln(output, prev)
		} else if !count && !dupsOnly && !uniqOnly {
			fmt.Fprintln(output, prev)
		}

		prev = line
		prevKey = key
		countNum = 1
	}

	// Output last group
	if !first {
		if dupsOnly && uniqOnly {
			// -d and -u together produce no output
		} else if count && !dupsOnly && !uniqOnly {
			fmt.Fprintf(output, "%d %s\n", countNum, prev)
		} else if dupsOnly && countNum > 1 {
			fmt.Fprintln(output, prev)
		} else if uniqOnly && countNum == 1 {
			fmt.Fprintln(output, prev)
		} else if !count && !dupsOnly && !uniqOnly {
			fmt.Fprintln(output, prev)
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
	breakSpaces := false
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		opt := paths[0]
		if opt == "--" {
			paths = paths[1:]
			break
		}
		if opt == "-s" {
			breakSpaces = true
			paths = paths[1:]
			continue
		}
		if opt == "-b" {
			// -b (count bytes) not yet implemented, just consume
			paths = paths[1:]
			continue
		}
		if opt == "-w" && len(paths) > 1 {
			width, _ = strconv.Atoi(paths[1])
			paths = paths[2:]
			continue
		}
		if strings.HasPrefix(opt, "-w") && len(opt) > 2 {
			width, _ = strconv.Atoi(opt[2:])
			paths = paths[1:]
			continue
		}
		// Handle combined options like -sw22
		if strings.HasPrefix(opt, "-") && len(opt) > 1 && opt[1] != 'w' {
			for _, flag := range opt[1:] {
				switch flag {
				case 's':
					breakSpaces = true
				case 'b':
					// -b: count bytes, no-op for now
				case 'w':
					// -w followed by number in same arg: handled above
				default:
					// might be a numeric width after flags
				}
			}
			// Check if there's a number after flags (e.g., -sw22)
			for i := 1; i < len(opt); i++ {
				if opt[i] >= '0' && opt[i] <= '9' {
					n, _ := strconv.Atoi(opt[i:])
					if n > 0 {
						width = n
					}
					break
				}
			}
			paths = paths[1:]
			continue
		}
		// Handle numeric width as option (e.g., fold -7)
		if n, err := strconv.Atoi(opt[1:]); err == nil && n > 0 {
			width = n
			paths = paths[1:]
			continue
		}
		paths = paths[1:]
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	// Adjust column for a character (tab advances to next 8-col boundary)
	adjustCol := func(col int, b byte) int {
		if b == '\t' {
			return col + 8 - col%8
		}
		return col + 1
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

		buf := make([]byte, 0, 4096)
		col := 0
		idx := 0
		for idx < len(data) {
			b := data[idx]
			if b == '\n' {
				buf = append(buf, '\n')
				os.Stdout.Write(buf)
				buf = buf[:0]
				col = 0
				idx++
				continue
			}
			newCol := adjustCol(col, b)
			if newCol > width && len(buf) > 0 {
				if breakSpaces {
					// Look backward for last blank (excluding current char)
					splitAt := -1
					for j := len(buf) - 1; j >= 0; j-- {
						if buf[j] == ' ' || buf[j] == '\t' {
							splitAt = j
							break
						}
					}
					if splitAt >= 0 {
						// Output up to and including the blank
						os.Stdout.Write(buf[:splitAt+1])
						os.Stdout.Write([]byte{'\n'})
						// Move rest (after blank) to front
						rest := make([]byte, len(buf)-(splitAt+1))
						copy(rest, buf[splitAt+1:])
						buf = buf[:0]
						// Recalculate column for the rest
						col = 0
						for _, rb := range rest {
							buf = append(buf, rb)
							col = adjustCol(col, rb)
						}
						// Re-process current character
						continue
					}
				}
				// No blank found (or -s not set): break at width boundary
				os.Stdout.Write(buf)
				os.Stdout.Write([]byte{'\n'})
				buf = buf[:0]
				col = 0
				continue
			}
			buf = append(buf, b)
			col = newCol
			idx++
		}
		if len(buf) > 0 {
			os.Stdout.Write(buf)
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
		s := string(data)
			if len(s) > 0 && s[len(s)-1] == '\n' {
				s = s[:len(s)-1]
			}
			for _, line := range strings.Split(s, "\n") {
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
	allBlanks := false
	firstOnly := false
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		opt := paths[0]
		if opt == "--" {
			paths = paths[1:]
			break
		}
		if opt == "-a" {
			allBlanks = true
			paths = paths[1:]
			continue
		}
		if opt == "-f" || opt == "--first-only" {
			firstOnly = true
			paths = paths[1:]
			continue
		}
		if strings.HasPrefix(opt, "-t") {
			if len(opt) > 2 {
				tabsize, _ = strconv.Atoi(opt[2:])
				paths = paths[1:]
			} else if len(paths) > 1 {
				tabsize, _ = strconv.Atoi(paths[1])
				paths = paths[2:]
			} else {
				paths = paths[1:]
			}
			allBlanks = true
			continue
		}
		// Unknown option, stop parsing
		break
	}
	// firstOnly overrides allBlanks
	if firstOnly {
		allBlanks = false
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
		s := string(data)
		hasTrailingNL := len(s) > 0 && s[len(s)-1] == '\n'
		if hasTrailingNL {
			s = s[:len(s)-1]
		}
		lines := strings.Split(s, "\n")
		for i, line := range lines {
			if i > 0 {
				fmt.Println()
			}
			fmt.Print(unexpandLine(line, tabsize, allBlanks))
		}
		if hasTrailingNL {
			fmt.Println()
		}
	}
	return exitCode
}

func unexpandLine(line string, tabsize int, convertBlanks bool) string {
	col := 0 // 0-indexed column
	var out strings.Builder
	spaces := 0
	leading := true

	flushSpaces := func() {
		if spaces == 0 {
			return
		}
		if leading || convertBlanks {
			// Convert spaces to tabs where possible
			startCol := col - spaces
			for spaces > 0 {
				nextTabStop := ((startCol + tabsize) / tabsize) * tabsize
				needed := nextTabStop - startCol
				if needed > 0 && needed <= spaces {
					out.WriteByte('\t')
					startCol += needed
					spaces -= needed
				} else {
					for i := 0; i < spaces; i++ {
						out.WriteByte(' ')
					}
					spaces = 0
				}
			}
		} else {
			for i := 0; i < spaces; i++ {
				out.WriteByte(' ')
			}
		}
		spaces = 0
	}

	for _, c := range line {
		if c == ' ' {
			spaces++
			col++
		} else if c == '\t' {
			// Tab: always combine with preceding spaces (tab subsumes spaces)
			tabStop := ((col + tabsize) / tabsize) * tabsize
			out.WriteByte('\t')
			spaces = 0
			col = tabStop
		} else {
			flushSpaces()
			leading = false
			out.WriteRune(c)
			col++
		}
	}
	// Flush trailing spaces
	for i := 0; i < spaces; i++ {
		out.WriteByte(' ')
	}
	return out.String()
}

func nlMain(args []string) int {
	bodyMode := byte('t') // default: number non-empty lines
	paths := args[1:]

	for len(paths) > 0 && strings.HasPrefix(paths[0], "-") {
		opt := paths[0]
		if opt == "-b" && len(paths) > 1 {
			if len(paths[1]) > 0 {
				bodyMode = paths[1][0]
			}
			paths = paths[2:]
		} else if strings.HasPrefix(opt, "-b") && len(opt) > 2 {
			bodyMode = opt[2]
			paths = paths[1:]
		} else if opt == "--" {
			paths = paths[1:]
			break
		} else {
			// unknown option, might be a filename starting with -
			break
		}
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
				fmt.Fprintf(os.Stderr, "gobox: nl: %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			defer f.Close()
			scanner = bufio.NewScanner(f)
		}
		lineNum := 1
		for scanner.Scan() {
			text := scanner.Text()
			number := (bodyMode == 'a') || (bodyMode == 't' && strings.TrimSpace(text) != "")
			if number {
				fmt.Printf("%6d\t%s\n", lineNum, text)
				lineNum++
			} else {
				fmt.Printf("%7s%s\n", "", text)
			}
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

	readFileOrStdin := func(name string) ([]byte, error) {
		if name == "-" {
			return io.ReadAll(os.Stdin)
		}
		return os.ReadFile(name)
	}

	data1, err := readFileOrStdin(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: comm: %s: %v\n", args[1], err)
		return 1
	}
	data2, err := readFileOrStdin(args[2])
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
			fmt.Printf("\t%s\n", lines2[j])
			j++
		case j >= len(lines2):
			fmt.Printf("%s\n", lines1[i])
			i++
		case lines1[i] < lines2[j]:
			fmt.Printf("%s\n", lines1[i])
			i++
		case lines1[i] > lines2[j]:
			fmt.Printf("\t%s\n", lines2[j])
			j++
		default:
			fmt.Printf("\t\t%s\n", lines1[i])
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
	sysv := false
	paths := args[1:]

	for len(paths) > 0 && paths[0][0] == '-' {
		opt := paths[0]
		for _, c := range opt[1:] {
			switch c {
			case 'r':
				sysv = false
			case 's':
				sysv = true
			default:
				fmt.Fprintf(os.Stderr, "gobox: sum: invalid option -- %c\n", c)
				return 1
			}
		}
		paths = paths[1:]
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
			fmt.Fprintf(os.Stderr, "gobox: sum: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		var s int
		if sysv {
			s = sysvSum(data)
		} else {
			s = bsdSum(data)
		}
		if path == "" {
			fmt.Printf("%d %d\n", s, (len(data)+511)/512)
		} else {
			showFilename := sysv || len(paths) > 1
			if showFilename {
				fmt.Printf("%d %d %s\n", s, (len(data)+511)/512, path)
			} else {
				fmt.Printf("%d %d\n", s, (len(data)+511)/512)
			}
		}
	}
	return exitCode
}

func sysvSum(data []byte) int {
	var s uint
	for _, b := range data {
		s += uint(b)
	}
	return int((s & 0xFFFF) + (s >> 16))
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

func sha3New() hash.Hash {
	return sha3.New256()
}
