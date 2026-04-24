package coreutils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gobox/applets"
)

func init() {
	applets.Register("grep", applets.AppletFunc(grepMain))
	applets.Register("egrep", applets.AppletFunc(egrepMain))
	applets.Register("fgrep", applets.AppletFunc(fgrepMain))
	applets.Register("find", applets.AppletFunc(findMain))
	applets.Register("sed", applets.AppletFunc(sedMain))
	applets.Register("diff", applets.AppletFunc(diffMain))
	applets.Register("patch", applets.AppletFunc(patchMain))
	applets.Register("cmp", applets.AppletFunc(cmpMain))
}

func grepMain(args []string) int {
	ignoreCase := false
	printLineNum := false
	reverse := false
	countOnly := false
	filesOnly := false
	filesWithoutMatch := false
	recursive := false
	wordRegexp := false
	lineRegexp := false
	quiet := false
	alwaysPrintFilename := false
	noFilename := false
	fixedStrings := false
	onlyMatching := false
	var patterns []string
	var filePatterns []string
	paths := []string{}
	hadError := false

	i := 1
loop:
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			i++
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			break
		}
		for j := 1; j < len(arg); j++ {
			c := arg[j]
			switch c {
			case 'i':
				ignoreCase = true
			case 'n':
				printLineNum = true
			case 'v':
				reverse = true
			case 'c':
				countOnly = true
			case 'l':
				filesOnly = true
			case 'L':
				filesWithoutMatch = true
			case 'r', 'R':
				recursive = true
			case 'w':
				wordRegexp = true
			case 'x':
				lineRegexp = true
			case 'q':
				quiet = true
			case 'H':
				alwaysPrintFilename = true
			case 'h':
				noFilename = true
			case 's':
				// suppress error messages
			case 'E':
				// extended regex (Go regexp is extended by default)
			case 'F':
				fixedStrings = true
			case 'o':
				onlyMatching = true
			case 'e':
				if j+1 < len(arg) {
					patterns = append(patterns, arg[j+1:])
					i++
					continue loop
				} else if i+1 < len(args) {
					i++
					patterns = append(patterns, args[i])
				}
				i++
				continue loop
			case 'f':
				if j+1 < len(arg) {
					filePatterns = append(filePatterns, arg[j+1:])
					i++
					continue loop
				} else if i+1 < len(args) {
					i++
					filePatterns = append(filePatterns, args[i])
				}
				i++
				continue loop
			}
		}
		i++
	}

	// Collect positional pattern (only if no -e or -f given)
	if len(patterns) == 0 && len(filePatterns) == 0 && i < len(args) {
		patterns = append(patterns, args[i])
		i++
	}

	// Read patterns from -f files
	var allPatterns []string
	for _, p := range patterns {
		allPatterns = append(allPatterns, strings.Split(p, "\n")...)
	}
	for _, f := range filePatterns {
		var data []byte
		var err error
		if f == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(f)
		}
		if err != nil {
			hadError = true
			continue
		}
		if len(data) > 0 {
			lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			allPatterns = append(allPatterns, lines...)
		}
	}

	// No pattern at all: match nothing (or everything with -v)
	noPattern := len(allPatterns) == 0

	// Build combined regexp
	var re *regexp.Regexp
	if !noPattern {
		searchPattern := strings.Join(allPatterns, "|")
		if fixedStrings {
			parts := make([]string, len(allPatterns))
			for i, p := range allPatterns {
				parts[i] = regexp.QuoteMeta(p)
			}
			searchPattern = strings.Join(parts, "|")
		}
		if wordRegexp {
			// Pattern consisting only of anchors matches nothing with -w
			stripped := strings.TrimPrefix(searchPattern, "^")
			stripped = strings.TrimSuffix(stripped, "$")
			if stripped == "" {
				searchPattern = `x\by` // never matches (Go RE2 doesn't support lookahead)
			} else {
				if !strings.HasPrefix(searchPattern, "^") {
					searchPattern = `\b` + searchPattern
				}
				if !strings.HasSuffix(searchPattern, "$") {
					searchPattern = searchPattern + `\b`
				}
			}
		}
		if lineRegexp {
			searchPattern = `^` + searchPattern + `$`
		}
		if ignoreCase {
			searchPattern = "(?i)" + searchPattern
		}
		var err error
		re, err = regexp.Compile(searchPattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: grep: invalid pattern: %s\n", err)
			return 1
		}
	}

	paths = args[i:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	// Handle recursive: if paths are directories, search them recursively
	if recursive {
		var expanded []string
		for _, p := range paths {
			if p == "" {
				expanded = append(expanded, "")
				continue
			}
			info, err := os.Stat(p)
			if err != nil {
				expanded = append(expanded, p)
				continue
			}
			if !info.IsDir() {
				expanded = append(expanded, p)
				continue
			}
			// Resolve symlinks to directories (filepath.Walk doesn't follow them)
			walkRoot := p
			if lstatInfo, lerr := os.Lstat(p); lerr == nil && lstatInfo.Mode()&os.ModeSymlink != 0 {
				if resolved, rerr := filepath.EvalSymlinks(p); rerr == nil {
					walkRoot = resolved
				}
			}
			filepath.Walk(walkRoot, func(fp string, fi os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if fi.IsDir() {
					return nil
				}
				// Map back to original path for display
				displayPath := fp
				if walkRoot != p {
					displayPath = filepath.Join(p, strings.TrimPrefix(fp, walkRoot))
				}
				expanded = append(expanded, displayPath)
				return nil
			})
		}
		paths = expanded
	}

	showFilename := len(paths) > 1 || alwaysPrintFilename || recursive
	if noFilename {
		showFilename = false
	}
	exitCode := 1
	// Track per-file match status (for -L support)
	type fileResult struct {
		matched bool
		count   int
	}
	fileResults := make(map[string]*fileResult)

	for _, path := range paths {
		var scanner *bufio.Scanner
		fname := path

		if path == "" || path == "-" {
			scanner = bufio.NewScanner(os.Stdin)
			fname = "(standard input)"
		} else {
			f, err := os.Open(path)
			if err != nil {
				hadError = true
				continue
			}
			defer f.Close()
			scanner = bufio.NewScanner(f)
		}

		lineNum := 1
		result := &fileResult{}
		fileResults[fname] = result
		for scanner.Scan() {
			line := scanner.Text()
			var matched bool
			if noPattern {
				matched = reverse
			} else {
				matched = re.MatchString(line)
				if reverse {
					matched = !matched
				}
			}
			if matched {
				result.count++
				result.matched = true
				exitCode = 0
				if filesOnly {
					fmt.Println(fname)
					break
				}
				if filesWithoutMatch {
					continue
				}
				if countOnly {
					continue
				}
				if quiet {
					return 0
				}
				if onlyMatching {
					matches := re.FindAllString(line, -1)
					for _, m := range matches {
						if m == "" {
							continue
						}
						if showFilename {
							if printLineNum {
								fmt.Printf("%s:%d:%s\n", fname, lineNum, m)
							} else {
								fmt.Printf("%s:%s\n", fname, m)
							}
						} else {
							if printLineNum {
								fmt.Printf("%d:%s\n", lineNum, m)
							} else {
								fmt.Println(m)
							}
						}
					}
					continue
				}
				if showFilename {
					if printLineNum {
						fmt.Printf("%s:%d:%s\n", fname, lineNum, line)
					} else {
						fmt.Printf("%s:%s\n", fname, line)
					}
				} else {
					if printLineNum {
						fmt.Printf("%d:%s\n", lineNum, line)
					} else {
						fmt.Println(line)
					}
				}
			}
			lineNum++
		}
		if countOnly {
			if showFilename {
				fmt.Printf("%s:%d\n", fname, result.count)
			} else {
				fmt.Printf("%d\n", result.count)
			}
		}
	}

	// -L: print files without matches
	if filesWithoutMatch {
		printed := false
		for _, path := range paths {
			fname := path
			if path == "" || path == "-" {
				fname = "(standard input)"
			}
			r := fileResults[fname]
			if r == nil || !r.matched {
				fmt.Println(fname)
				printed = true
			}
		}
		if printed {
			exitCode = 0
		} else {
			exitCode = 1
		}
	}

	if hadError {
		return 2
	}
	return exitCode
}

func egrepMain(args []string) int {
	// egrep is grep -E
	newArgs := []string{args[0], "-E"}
	newArgs = append(newArgs, args[1:]...)
	return grepMain(newArgs)
}

func fgrepMain(args []string) int {
	// fgrep is grep -F
	newArgs := []string{args[0], "-F"}
	newArgs = append(newArgs, args[1:]...)
	return grepMain(newArgs)
}

func findMain(args []string) int {
	paths := []string{"."}
	exprStart := 1

	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		paths = []string{args[1]}
		exprStart = 2
	}

	type predicate struct {
		name string
		val  string
	}
	var predicates []predicate
	var execCmd []string
	execAction := ""
	execFound := false
	execPlus := false

	for i := exprStart; i < len(args); i++ {
		if args[i] == "-exec" || args[i] == "-ok" {
			execAction = args[i]
			execFound = true
			i++
			for i < len(args) && args[i] != ";" && args[i] != "+" {
				if args[i] == "{}" {
					execCmd = append(execCmd, "{}")
				} else {
					execCmd = append(execCmd, args[i])
				}
				i++
			}
			if i < len(args) && args[i] == "+" {
				execPlus = true
			}
			continue
		}
		if args[i] == "-delete" {
			predicates = append(predicates, predicate{"delete", ""})
			continue
		}
		if args[i] == "-empty" {
			predicates = append(predicates, predicate{"empty", ""})
			continue
		}
		switch args[i] {
		case "-name":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"name", args[i+1]})
				i++
			}
		case "-type":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"type", args[i+1]})
				i++
			}
		case "-size":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"size", args[i+1]})
				i++
			}
		case "-maxdepth":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"maxdepth", args[i+1]})
				i++
			}
		case "-mindepth":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"mindepth", args[i+1]})
				i++
			}
		case "-mtime":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"mtime", args[i+1]})
				i++
			}
		case "-print":
			predicates = append(predicates, predicate{"print", ""})
		case "-user":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"user", args[i+1]})
				i++
			}
		case "-group":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"group", args[i+1]})
				i++
			}
		case "-perm":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"perm", args[i+1]})
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				predicates = append(predicates, predicate{"path", args[i]})
			}
		}
	}

	maxDepth := -1
	minDepth := -1
	for _, p := range predicates {
		if p.name == "maxdepth" {
			fmt.Sscanf(p.val, "%d", &maxDepth)
		}
		if p.name == "mindepth" {
			fmt.Sscanf(p.val, "%d", &minDepth)
		}
	}

	hasAction := false
	if execFound {
		hasAction = true
	} else {
		for _, p := range predicates {
			if p.name == "print" || p.name == "delete" {
				hasAction = true
				break
			}
		}
	}

	exitCode := 0
	for _, root := range paths {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			depth := 0
			if path != root && len(path) >= len(root) {
				depth = strings.Count(path[len(root):], string(filepath.Separator))
			}
			if maxDepth >= 0 && depth > maxDepth {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if minDepth >= 0 && depth < minDepth {
				return nil
			}

			match := true
			for _, p := range predicates {
				switch p.name {
				case "name":
					m, err := filepath.Match(p.val, info.Name())
					if err != nil || !m {
						match = false
					}
				case "type":
					switch p.val {
					case "f":
						match = !info.IsDir()
					case "d":
						match = info.IsDir()
					case "l":
						match = info.Mode()&os.ModeSymlink != 0
					case "s":
						match = info.Mode()&os.ModeSocket != 0
					default:
						match = false
					}
				case "size":
					match = matchSize(p.val, info.Size())
				case "empty":
					match = info.Size() == 0
				case "user":
					match = matchUser(p.val)
				case "print", "":
					// always matches
				case "delete":
					if match {
						os.RemoveAll(path)
					}
				case "path":
					if path != p.val {
						match = false
					}
				}
			}

			if match && execFound && len(execCmd) > 0 {
				cmdArgs := make([]string, len(execCmd))
				for i, a := range execCmd {
					if a == "{}" {
						cmdArgs[i] = path
					} else {
						cmdArgs[i] = a
					}
				}
				if execAction == "-ok" {
					for _, a := range cmdArgs {
						fmt.Fprintf(os.Stderr, "%s ", a)
					}
					fmt.Fprintf(os.Stderr, "?")
					var resp string
					fmt.Scanf("%s", &resp)
					if resp != "y" && resp != "Y" {
						return nil
					}
				}
				cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil && execPlus {
					exitCode = 1
				}
			}

			if match && !hasAction {
				displayPath := path
				if root == "." && !strings.HasPrefix(path, "./") && path != "." {
					displayPath = "./" + path
				}
				fmt.Println(displayPath)
			} else if match && hasAction {
				for _, p := range predicates {
					if p.name == "print" {
						displayPath := path
						if root == "." && !strings.HasPrefix(path, "./") && path != "." {
							displayPath = "./" + path
						}
						fmt.Println(displayPath)
					}
				}
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: find: %v\n", err)
			exitCode = 1
		}
	}
	return exitCode
}

func matchSize(pattern string, size int64) bool {
	if pattern == "" {
		return true
	}
	if strings.HasPrefix(pattern, "+") {
		n, _ := strconv.ParseInt(pattern[1:], 10, 64)
		return size > n
	}
	if strings.HasPrefix(pattern, "-") {
		n, _ := strconv.ParseInt(pattern[1:], 10, 64)
		return size < n
	}
	n, _ := strconv.ParseInt(pattern, 10, 64)
	return size == n
}

func matchUser(name string) bool {
	// Always match if we can't verify
	return true
}

// sedPatternToRE2 converts POSIX basic regex \( \) to Go RE2 ( )
func sedPatternToRE2(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			if s[i+1] == '(' || s[i+1] == ')' {
				b.WriteByte(s[i+1])
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// sedReplacementToRE2 converts \1 \2 etc. to $1 $2 for Go regexp replacement
func sedReplacementToRE2(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			b.WriteByte('$')
			b.WriteByte(s[i+1])
			i += 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func sedMain(args []string) int {
	var script string
	var inPlace string
	paths := []string{}

	i := 1
	for i < len(args) {
		arg := args[i]
		if arg == "-e" && i+1 < len(args) {
			script = args[i+1]
			i += 2
		} else if arg == "-i" {
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				inPlace = args[i+1]
				i += 2
			} else {
				inPlace = ""
				i++
			}
		} else if !strings.HasPrefix(arg, "-") {
			paths = args[i:]
			break
		} else {
			i++
		}
	}

	if script == "" && len(paths) > 0 {
		script = paths[0]
		paths = paths[1:]
	}

	if script == "" {
		fmt.Fprintln(os.Stderr, "gobox: sed: missing script")
		return 1
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	// Parse sed script: s/old/new/flags
	var searchStr, replaceStr string
	global := false
	if strings.HasPrefix(script, "s/") {
		parts := strings.SplitN(script[2:], "/", 3)
		if len(parts) >= 2 {
			searchStr = parts[0]
			replaceStr = parts[1]
			if len(parts) >= 3 {
				global = strings.Contains(parts[2], "g")
			}
		}
	}

	// Convert POSIX-style \( \) to Go RE2 ( ), and \1 \2 to $1 $2
	searchStr = sedPatternToRE2(searchStr)
	replaceStr = sedReplacementToRE2(replaceStr)

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
			fmt.Fprintf(os.Stderr, "gobox: sed: %s: %v\n", path, err)
			exitCode = 1
			continue
		}

		re, err := regexp.Compile("(?m)" + searchStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: sed: invalid pattern: %s\n", err)
			return 1
		}

		result := ""
		if global {
			result = re.ReplaceAllString(string(data), replaceStr)
		} else {
			// sed processes line by line, replacing first match per line
			lines := strings.SplitAfter(string(data), "\n")
			for _, line := range lines {
				loc := re.FindStringIndex(line)
				if loc != nil {
					before := line[:loc[0]]
					match := re.ReplaceAllString(line[loc[0]:loc[1]], replaceStr)
					after := line[loc[1]:]
					result += before + match + after
				} else {
					result += line
				}
			}
		}

		if inPlace != "" {
			if inPlace == "" {
				os.WriteFile(path, []byte(result), 0644)
			} else {
				os.Rename(path, path+inPlace)
				os.WriteFile(path, []byte(result), 0644)
			}
		} else {
			fmt.Print(result)
		}
	}
	return exitCode
}

func diffMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: diff: missing operand")
		return 1
	}

	unified := false
	quiet := false
	ignoreWS := false
	ignoreBlankLines := false
	fileArgs := []string{}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "-" {
			fileArgs = append(fileArgs, arg)
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			fileArgs = append(fileArgs, arg)
			continue
		}
		// Parse combined flags like -ur, -qN
		for _, c := range arg[1:] {
			switch c {
			case 'q':
				quiet = true
			case 'u':
				unified = true
			case 'r':
				// recursive dir comparison (handled by diffDirs)
			case 'N':
				// treat absent files as empty (handled by diffDirs)
			case 'b':
				ignoreWS = true
			case 'B':
				ignoreBlankLines = true
			}
		}
	}

	if len(fileArgs) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: diff: missing operand")
		return 1
	}

	var stdinCache []byte
	readFileOrStdin := func(name string) ([]byte, error) {
		if name == "-" {
			if stdinCache != nil {
				return stdinCache, nil
			}
			var err error
			stdinCache, err = io.ReadAll(os.Stdin)
			return stdinCache, err
		}
		return os.ReadFile(name)
	}

	// diff - - means compare stdin with itself, identical
	if len(fileArgs) >= 2 && fileArgs[0] == "-" && fileArgs[1] == "-" {
		return 0
	}

	info1, err1 := os.Stat(fileArgs[0])
	info2, err2 := os.Stat(fileArgs[1])

	if err1 == nil && err2 == nil && info1.IsDir() && info2.IsDir() {
		return diffDirs(fileArgs[0], fileArgs[1], unified, quiet, ignoreWS, ignoreBlankLines)
	}

	// Handle dir/file comparison: diff dir file -> compare dir/basename(file) with file
	if err1 == nil && info1.IsDir() && err2 == nil && !info2.IsDir() {
		fileArgs[0] = filepath.Join(fileArgs[0], filepath.Base(fileArgs[1]))
		info1, err1 = os.Stat(fileArgs[0])
	}
	if err1 == nil && !info1.IsDir() && err2 == nil && info2.IsDir() {
		fileArgs[1] = filepath.Join(fileArgs[1], filepath.Base(fileArgs[0]))
		info2, err2 = os.Stat(fileArgs[1])
	}

	data1, err := readFileOrStdin(fileArgs[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: diff: %s: %v\n", fileArgs[0], err)
		return 1
	}
	data2, err := readFileOrStdin(fileArgs[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: diff: %s: %v\n", fileArgs[1], err)
		return 1
	}

	hasNewline1 := len(data1) == 0 || data1[len(data1)-1] == '\n'
	hasNewline2 := len(data2) == 0 || data2[len(data2)-1] == '\n'

	s1 := string(data1)
	if len(s1) > 0 && s1[len(s1)-1] == '\n' {
		s1 = s1[:len(s1)-1]
	}
	lines1 := strings.Split(s1, "\n")
	if len(s1) == 0 {
		lines1 = nil
	}
	s2 := string(data2)
	if len(s2) > 0 && s2[len(s2)-1] == '\n' {
		s2 = s2[:len(s2)-1]
	}
	lines2 := strings.Split(s2, "\n")
	if len(s2) == 0 {
		lines2 = nil
	}

	diff := buildDiffWithOpts(lines1, lines2, ignoreWS)

	if len(diff) == 0 || allEqual(diff) {
		return 0
	}

	if quiet {
		if ignoreBlankLines {
			hunks := buildHunks(diff, true)
			if len(hunks) == 0 {
				return 0
			}
		}
		fmt.Printf("Files %s and %s differ\n", fileArgs[0], fileArgs[1])
	} else if unified {
		hunks := buildHunks(diff, ignoreBlankLines)
		if len(hunks) == 0 {
			return 0
		}
		fmt.Printf("--- %s\n+++ %s\n", fileArgs[0], fileArgs[1])
		printHunks(hunks, hasNewline1, hasNewline2)
	} else {
		for _, d := range diff {
			switch d.op {
			case '-':
				fmt.Printf("< %s\n", d.text)
			case '+':
				fmt.Printf("> %s\n", d.text)
			}
		}
	}
	return 1
}

type diffLine struct {
	op   byte // '=', '-', '+'
	text string
}

func allEqual(diff []diffLine) bool {
	for _, d := range diff {
		if d.op != '=' {
			return false
		}
	}
	return true
}

func linesEqual(a, b string, ignoreWS bool) bool {
	if !ignoreWS {
		return a == b
	}
	// Normalize whitespace: collapse runs of spaces/tabs, trim trailing
	norm := func(s string) string {
		inSpace := false
		var result strings.Builder
		for _, ch := range s {
			if ch == ' ' || ch == '\t' {
				if !inSpace {
					result.WriteByte(' ')
					inSpace = true
				}
			} else {
				result.WriteRune(ch)
				inSpace = false
			}
		}
		// Trim trailing space
		s2 := result.String()
		if len(s2) > 0 && s2[len(s2)-1] == ' ' {
			s2 = s2[:len(s2)-1]
		}
		return s2
	}
	return norm(a) == norm(b)
}

func buildDiff(a, b []string) []diffLine {
	return buildDiffWithOpts(a, b, false)
}

func buildDiffWithOpts(a, b []string, ignoreWS bool) []diffLine {
	// LCS-based diff
	m, n := len(a), len(b)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if linesEqual(a[i-1], b[j-1], ignoreWS) {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else {
				if lcs[i-1][j] > lcs[i][j-1] {
					lcs[i][j] = lcs[i-1][j]
				} else {
					lcs[i][j] = lcs[i][j-1]
				}
			}
		}
	}

	var result []diffLine
	i, j := m, n
	var temp []diffLine
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && linesEqual(a[i-1], b[j-1], ignoreWS) {
			temp = append(temp, diffLine{'=', a[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			temp = append(temp, diffLine{'+', b[j-1]})
			j--
		} else if i > 0 {
			temp = append(temp, diffLine{'-', a[i-1]})
			i--
		}
	}

	for k := len(temp) - 1; k >= 0; k-- {
		result = append(result, temp[k])
	}
	return result
}

type hunk struct {
	oldStart, oldCount int
	newStart, newCount int
	lines              []diffLine
}

func buildHunks(diff []diffLine, ignoreBlankLines bool) []hunk {
	var hunks []hunk
	i := 0
	n := len(diff)
	for i < n {
		for i < n && diff[i].op == '=' {
			i++
		}
		if i >= n {
			break
		}
		hstart := i - 3
		if hstart < 0 {
			hstart = 0
		}
		j := i
		for j < n && diff[j].op != '=' {
			j++
		}
		hend := j + 3
		if hend > n {
			hend = n
		}
		oldNum, newNum := 1, 1
		for k := 0; k < hstart; k++ {
			if diff[k].op == '=' || diff[k].op == '-' {
				oldNum++
			}
			if diff[k].op == '=' || diff[k].op == '+' {
				newNum++
			}
		}
		h := hunk{oldStart: oldNum, newStart: newNum}
		for k := hstart; k < hend; k++ {
			h.lines = append(h.lines, diff[k])
			if diff[k].op == '=' || diff[k].op == '-' {
				h.oldCount++
			}
			if diff[k].op == '=' || diff[k].op == '+' {
				h.newCount++
			}
		}
		if ignoreBlankLines {
			allBlank := true
			for _, d := range h.lines {
				if d.op != '=' && strings.TrimSpace(d.text) != "" {
					allBlank = false
					break
				}
			}
			if allBlank {
				i = hend
				continue
			}
		}
		hunks = append(hunks, h)
		i = hend
	}
	return hunks
}

func printHunks(hunks []hunk, hasNewline1, hasNewline2 bool) {
	for _, h := range hunks {
		newStart := h.newStart
		if h.newCount == 0 {
			newStart = 0
		}
		oldStart := h.oldStart
		if h.oldCount == 0 {
			oldStart = 0
		}
		if h.oldCount == 1 && h.newCount == 1 {
			fmt.Printf("@@ -%d +%d @@\n", oldStart, newStart)
		} else if h.oldCount == 1 {
			fmt.Printf("@@ -%d +%d,%d @@\n", oldStart, newStart, h.newCount)
		} else if h.newCount == 1 {
			fmt.Printf("@@ -%d,%d +%d @@\n", oldStart, h.oldCount, newStart)
		} else {
			fmt.Printf("@@ -%d,%d +%d,%d @@\n", oldStart, h.oldCount, newStart, h.newCount)
		}
		for _, d := range h.lines {
			switch d.op {
			case '=':
				fmt.Printf(" %s\n", d.text)
			case '-':
				fmt.Printf("-%s\n", d.text)
			case '+':
				fmt.Printf("+%s\n", d.text)
			}
		}
		if !hasNewline2 && len(h.lines) > 0 {
			last := h.lines[len(h.lines)-1]
			if last.op == '+' || last.op == '=' {
				fmt.Println("\\ No newline at end of file")
			}
		}
		if !hasNewline1 && len(h.lines) > 0 {
			last := h.lines[len(h.lines)-1]
			if last.op == '-' || last.op == '=' {
				fmt.Println("\\ No newline at end of file")
			}
		}
	}
}

func diffTwoFiles(path1, path2 string, unified, quiet bool, ignoreWS, ignoreBlankLines bool, displayPath1, displayPath2 string) int {
	data1, err := os.ReadFile(path1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: diff: %s: %v\n", path1, err)
		return 1
	}
	data2, err := os.ReadFile(path2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: diff: %s: %v\n", path2, err)
		return 1
	}

	hasNewline1 := len(data1) == 0 || data1[len(data1)-1] == '\n'
	hasNewline2 := len(data2) == 0 || data2[len(data2)-1] == '\n'

	s1 := string(data1)
	if len(s1) > 0 && s1[len(s1)-1] == '\n' {
		s1 = s1[:len(s1)-1]
	}
	lines1 := strings.Split(s1, "\n")
	if len(s1) == 0 {
		lines1 = nil
	}
	s2 := string(data2)
	if len(s2) > 0 && s2[len(s2)-1] == '\n' {
		s2 = s2[:len(s2)-1]
	}
	lines2 := strings.Split(s2, "\n")
	if len(s2) == 0 {
		lines2 = nil
	}

	diff := buildDiffWithOpts(lines1, lines2, ignoreWS)

	if len(diff) == 0 || allEqual(diff) {
		return 0
	}

	if quiet {
		if ignoreBlankLines {
			hunks := buildHunks(diff, true)
			if len(hunks) == 0 {
				return 0
			}
		}
		if displayPath1 != "" { fmt.Printf("Files %s and %s differ\n", displayPath1, displayPath2) } else { fmt.Printf("Files %s and %s differ\n", path1, path2) }
	} else if unified {
		hunks := buildHunks(diff, ignoreBlankLines)
		if len(hunks) == 0 {
			return 0
		}
		if displayPath1 != "" { fmt.Printf("--- %s\n+++ %s\n", displayPath1, displayPath2) } else { fmt.Printf("--- %s\n+++ %s\n", path1, path2) }
		printHunks(hunks, hasNewline1, hasNewline2)
	} else {
		for _, d := range diff {
			switch d.op {
			case '-':
				fmt.Printf("< %s\n", d.text)
			case '+':
				fmt.Printf("> %s\n", d.text)
			}
		}
	}
	return 1
}

func displayPath(dir, name string) string {
	if strings.HasSuffix(dir, "/") {
		return dir + name
	}
	return dir + "/" + name
}

func diffDirs(dir1, dir2 string, unified, quiet bool, ignoreWS, ignoreBlankLines bool) int {
	exitCode := 0

	entries1, err := os.ReadDir(dir1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: diff: %s: %v\n", dir1, err)
		return 1
	}

	entries2, err := os.ReadDir(dir2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: diff: %s: %v\n", dir2, err)
		return 1
	}

	fileSet := make(map[string]bool)
	for _, e := range entries1 {
		fileSet[e.Name()] = true
	}
	for _, e := range entries2 {
		fileSet[e.Name()] = true
	}

	names := make([]string, 0, len(fileSet))
	for name := range fileSet {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		path1 := filepath.Join(dir1, name)
		path2 := filepath.Join(dir2, name)

		info1, err1 := os.Stat(path1)
		info2, err2 := os.Stat(path2)

		// Case: one side is a directory, the other is a non-regular file
		if err1 == nil && info1.IsDir() && err2 == nil && !info2.IsDir() && !info2.Mode().IsRegular() {
			fmt.Printf("Only in %s: %s\n", dir2, name)
			exitCode = 1
			continue
		}
		if err2 == nil && info2.IsDir() && err1 == nil && !info1.IsDir() && !info1.Mode().IsRegular() {
			fmt.Printf("Only in %s: %s\n", dir1, name)
			exitCode = 1
			continue
		}

		// Non-regular files (fifos, sockets, etc.) - skip even if only in one dir
		if err1 == nil && !info1.IsDir() && !info1.Mode().IsRegular() {
			fmt.Printf("File %s is not a regular file or directory and was skipped\n", path1)
			exitCode = 1
			continue
		}
		if err2 == nil && !info2.IsDir() && !info2.Mode().IsRegular() {
			fmt.Printf("File %s is not a regular file or directory and was skipped\n", path2)
			exitCode = 1
			continue
		}

		// Files only in one directory
		if os.IsNotExist(err1) {
			fmt.Printf("Only in %s: %s\n", dir2, name)
			exitCode = 1
			continue
		}
		if os.IsNotExist(err2) {
			fmt.Printf("Only in %s: %s\n", dir1, name)
			exitCode = 1
			continue
		}

		if info1.IsDir() && info2.IsDir() {
			ec := diffDirs(path1, path2, unified, quiet, ignoreWS, ignoreBlankLines)
			if ec != 0 {
				exitCode = 1
			}
		} else if info1.IsDir() || info2.IsDir() {
			fmt.Printf("File %s is a directory while file %s is a regular file\n", path1, path2)
			exitCode = 1
		} else {
			disp1 := displayPath(dir1, name)
			disp2 := displayPath(dir2, name)
			ec := diffTwoFiles(path1, path2, unified, quiet, ignoreWS, ignoreBlankLines, disp1, disp2)
			if ec != 0 {
				exitCode = 1
			}
		}
	}
	return exitCode
}

func patchMain(args []string) int {
	reverse := false
	noReverse := false
	stripLevel := 0
	patchArgs := args[1:]

	// Parse flags and positional args
	var patchFile string
	var targetFile string
	i := 0
	for ; i < len(patchArgs); i++ {
		a := patchArgs[i]
		if a == "--" {
			i++
			break
		}
		if !strings.HasPrefix(a, "-") {
			break
		}
		switch {
		case a == "-R":
			reverse = true
		case a == "-N":
			noReverse = true
		case a == "-p0":
			stripLevel = 0
		case a == "-p1":
			stripLevel = 1
		case strings.HasPrefix(a, "-p"):
			if n, err := strconv.Atoi(a[2:]); err == nil && n >= 0 {
				stripLevel = n
			}
		case a == "-i":
			// -i is ignored (we read from stdin by default)
		default:
			// Unknown flag, ignore
		}
	}

	// Remaining args: [target_file [patch_file]]
	remaining := patchArgs[i:]
	if len(remaining) >= 1 {
		targetFile = remaining[0]
	}
	if len(remaining) >= 2 {
		patchFile = remaining[1]
	}

	// Read patch input
	var patchData []byte
	var err error
	if patchFile != "" {
		patchData, err = os.ReadFile(patchFile)
	} else {
		patchData, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: patch: %v\n", err)
		return 1
	}

	// Parse patches grouped by target file
	type filePatch struct {
		oldPath string
		newPath string
		hunks   []patchHunk
	}

	patches := parseUnifiedDiff(string(patchData))
	if len(patches) == 0 {
		return 1
	}

	exitCode := 0
	for _, fp := range patches {
		// Determine the target path
		path := fp.newPath
		if path == "" || path == "/dev/null" {
			path = fp.oldPath
		}
		// Apply -p strip
		if stripLevel > 0 {
			for s := 0; s < stripLevel; s++ {
				idx := strings.Index(path, "/")
				if idx < 0 {
					break
				}
				path = path[idx+1:]
			}
			// Strip leading slashes
			path = strings.TrimLeft(path, "/")
		}
		if path == "/dev/null" {
			continue
		}

		// Override target if specified on command line
		applyPath := path
		if targetFile != "" {
			applyPath = targetFile
		}

		// Check if this is a new file creation
		isNewFile := fp.oldPath == "/dev/null"

		if isNewFile {
			fmt.Printf("creating %s\n", path)
			content := ""
			ok := applyHunks(&content, fp.hunks, reverse, noReverse)
			if !ok {
				exitCode = 1
			}
			err := os.WriteFile(applyPath, []byte(content), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: patch: can't create '%s': %v\n", path, err)
				exitCode = 1
			}
			continue
		}

		// Use applyPath for the message when targetFile was specified
		msgPath := path
		if targetFile != "" {
			msgPath = applyPath
		}
		fmt.Printf("patching file %s\n", msgPath)

		// Read the target file
		data, err := os.ReadFile(applyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "patch: can't open '%s': %s\n", path, "No such file or directory")
			exitCode = 1
			continue
		}
		content := string(data)

		ok := applyHunks(&content, fp.hunks, reverse, noReverse)
		if !ok {
			exitCode = 1
		}

		// Write back
		err = os.WriteFile(applyPath, []byte(content), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: patch: can't write '%s': %v\n", applyPath, err)
			exitCode = 1
		}
	}

	return exitCode
}

type patchHunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []string // with leading char: ' ', '+', '-'
}

type filePatch struct {
	oldPath string
	newPath string
	hunks   []patchHunk
}

func parseUnifiedDiff(data string) []filePatch {
	var patches []filePatch
	var cur *filePatch
	lines := strings.Split(data, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "--- ") {
			// Start a new file entry
			oldPath := strings.TrimSpace(line[4:])
			// Remove timestamp after tab
			if idx := strings.Index(oldPath, "\t"); idx >= 0 {
				oldPath = oldPath[:idx]
			}
			// Read the next line for +++
			if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "+++ ") {
				newPath := strings.TrimSpace(lines[i+1][4:])
				if idx := strings.Index(newPath, "\t"); idx >= 0 {
					newPath = newPath[:idx]
				}
				cur = &filePatch{oldPath: oldPath, newPath: newPath}
				patches = append(patches, filePatch{})
				patches[len(patches)-1] = *cur
				cur = &patches[len(patches)-1]
				i++
			}
		} else if strings.HasPrefix(line, "@@") && cur != nil {
			// Parse hunk header: @@ -start,count +start,count @@
			hdr := line
			// Find the second @@
			endIdx := strings.Index(hdr[2:], "@@")
			if endIdx < 0 {
				continue
			}
			body := strings.TrimSpace(hdr[2 : endIdx+2])
			parts := strings.SplitN(body, " ", 2)
			if len(parts) < 2 {
				continue
			}
			oldPart := parts[0] // -start,count
			newPart := parts[1] // +start,count

			var oldStart, oldCount, newStart, newCount int
			oldCount = 1
			newCount = 1

			if o := strings.TrimPrefix(oldPart, "-"); o != oldPart {
				if commaIdx := strings.Index(o, ","); commaIdx >= 0 {
					oldStart, _ = strconv.Atoi(o[:commaIdx])
					oldCount, _ = strconv.Atoi(o[commaIdx+1:])
				} else {
					oldStart, _ = strconv.Atoi(o)
				}
			}
			if n := strings.TrimPrefix(newPart, "+"); n != newPart {
				if commaIdx := strings.Index(n, ","); commaIdx >= 0 {
					newStart, _ = strconv.Atoi(n[:commaIdx])
					newCount, _ = strconv.Atoi(n[commaIdx+1:])
				} else {
					newStart, _ = strconv.Atoi(n)
				}
			}

			// Read hunk body lines
			var hunkLines []string
			i++
			for ; i < len(lines); i++ {
				hl := lines[i]
				if strings.HasPrefix(hl, "@@") || strings.HasPrefix(hl, "---") || strings.HasPrefix(hl, "-- ") {
					i--
					break
				}
				// Empty lines inside a hunk are context lines (empty content)
				if hl == "" || strings.HasPrefix(hl, " ") || strings.HasPrefix(hl, "+") || strings.HasPrefix(hl, "-") || strings.HasPrefix(hl, "\\") {
					hunkLines = append(hunkLines, hl)
				} else {
					i--
					break
				}
			}
			// Trim trailing empty lines from hunk (caused by trailing newline in input)
			for len(hunkLines) > 0 && hunkLines[len(hunkLines)-1] == "" {
				hunkLines = hunkLines[:len(hunkLines)-1]
			}

			hunk := patchHunk{
				oldStart: oldStart,
				oldCount: oldCount,
				newStart: newStart,
				newCount: newCount,
				lines:    hunkLines,
			}
			cur.hunks = append(cur.hunks, hunk)
		}
	}

	return patches
}

func applyHunks(content *string, hunks []patchHunk, reverse bool, noReverse bool) bool {
	lines := strings.Split(*content, "\n")
	// Remove trailing empty line from split
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	allOk := true

	for hunkIdx, hunk := range hunks {
		// Build the old lines (what we're looking for) and new lines (what to replace with)
		var oldLines, newLines []string
		for _, hl := range hunk.lines {
			if hl == "" || hl[0] == '\\' {
				continue
			}
			switch hl[0] {
			case ' ':
				oldLines = append(oldLines, hl[1:])
				newLines = append(newLines, hl[1:])
			case '-':
				oldLines = append(oldLines, hl[1:])
			case '+':
				newLines = append(newLines, hl[1:])
			}
		}

		if reverse {
			oldLines, newLines = newLines, oldLines
		}

		// Handle new file creation (hunk with oldCount == 0)
		if hunk.oldCount == 0 {
			lines = append(lines, newLines...)
			continue
		}

		// Find the starting position
		startPos := hunk.oldStart - 1
		if startPos < 0 {
			startPos = 0
		}
		if startPos >= len(lines) {
			startPos = len(lines) - 1
		}

		// Try forward match (context lines match)
		forwardMatch := findMatch(lines, oldLines, startPos)
		// Try reverse match (new lines match as if already applied)
		reverseMatch := findMatch(lines, newLines, startPos)

		if reverseMatch >= 0 && (forwardMatch < 0 || forwardMatch == reverseMatch) {
			// Patch appears to already be applied
			atLine := reverseMatch + len(newLines) + 1
			if !noReverse && !reverse {
				fmt.Fprintf(os.Stderr, "Possibly reversed hunk %d at %d\n", hunkIdx+1, atLine)
			}
			if !noReverse {
				fmt.Printf("Hunk %d FAILED %d/%d.\n", hunkIdx+1, hunkIdx+1, len(hunks))
				for _, hl := range hunk.lines {
					fmt.Println(hl)
				}
				allOk = false
			}
			// For -N (noReverse), silently skip
			continue
		}

		if forwardMatch >= 0 {
			// Apply the hunk: replace oldLines at matchPos with newLines
			newLinesSlice := make([]string, 0, len(lines)-len(oldLines)+len(newLines))
			newLinesSlice = append(newLinesSlice, lines[:forwardMatch]...)
			newLinesSlice = append(newLinesSlice, newLines...)
			newLinesSlice = append(newLinesSlice, lines[forwardMatch+len(oldLines):]...)
			lines = newLinesSlice
		} else {
			// No match found
			fmt.Printf("Hunk %d FAILED %d/%d.\n", hunkIdx+1, hunkIdx+1, len(hunks))
			for _, hl := range hunk.lines {
				fmt.Println(hl)
			}
			allOk = false
		}
	}

	*content = strings.Join(lines, "\n")
	return allOk
}

func findMatch(lines, pattern []string, startPos int) int {
	if len(pattern) == 0 {
		return startPos
	}

	// Search from startPos forward, then backward
	maxSearch := 20 // search up to 20 lines away
	searchRange := []int{0}
	for d := 1; d <= maxSearch; d++ {
		searchRange = append(searchRange, d, -d)
	}

	for _, offset := range searchRange {
		pos := startPos + offset
		if pos < 0 || pos+len(pattern) > len(lines) {
			continue
		}
		match := true
		for j, pl := range pattern {
			if lines[pos+j] != pl {
				match = false
				break
			}
		}
		if match {
			return pos
		}
	}

	return -1
}

func cmpMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: cmp: missing operand")
		return 1
	}

	data1, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: cmp: %s: %v\n", args[1], err)
		return 1
	}
	data2, err := os.ReadFile(args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: cmp: %s: %v\n", args[2], err)
		return 1
	}

	minLen := len(data1)
	if len(data2) < minLen {
		minLen = len(data2)
	}

	for i := 0; i < minLen; i++ {
		if data1[i] != data2[i] {
			fmt.Printf("%s %s differ: char %d\n", args[1], args[2], i+1)
			return 1
		}
	}

	if len(data1) != len(data2) {
		fmt.Printf("cmp: EOF on %s\n", args[2])
		return 1
	}

	return 0
}
