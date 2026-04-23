package coreutils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
	pattern := ""
	paths := []string{}

	i := 1
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			i++
			break
		}
		if strings.HasPrefix(arg, "-") {
			for _, c := range arg[1:] {
				switch c {
				case 'i':
					ignoreCase = true
				case 'n':
					printLineNum = true
				case 'v':
					reverse = true
				case 'E':
					// extended regex - same behavior
				case 'F':
					// fixed strings - same behavior
				}
			}
			i++
		} else {
			break
		}
	}

	if pattern == "" && i < len(args) {
		pattern = args[i]
		i++
	}

	if pattern == "" {
		fmt.Fprintln(os.Stderr, "gobox: grep: missing pattern")
		return 1
	}

	paths = args[i:]
	if len(paths) == 0 {
		paths = []string{""}
	}

	if ignoreCase {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: grep: invalid pattern: %s\n", err)
		return 1
	}

	multipleFiles := len(paths) > 1
	exitCode := 1

	for _, path := range paths {
		var scanner *bufio.Scanner
		fname := path

		if path == "" {
			scanner = bufio.NewScanner(os.Stdin)
			fname = "(standard input)"
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: grep: %s: %v\n", path, err)
				continue
			}
			defer f.Close()
			scanner = bufio.NewScanner(f)
		}

		lineNum := 1
		for scanner.Scan() {
			line := scanner.Text()
			matched := re.MatchString(line)
			if reverse {
				matched = !matched
			}
			if matched {
				exitCode = 0
				if multipleFiles {
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

	// Parse expressions
	type predicate struct {
		name string
		val  string
	}
	var predicates []predicate

	for i := exprStart; i < len(args); i++ {
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
		case "-mtime":
			if i+1 < len(args) {
				predicates = append(predicates, predicate{"mtime", args[i+1]})
				i++
			}
		case "-print":
			predicates = append(predicates, predicate{"print", ""})
		default:
			if !strings.HasPrefix(args[i], "-") {
				predicates = append(predicates, predicate{"path", args[i]})
			}
		}
	}

	maxDepth := -1
	for _, p := range predicates {
		if p.name == "maxdepth" {
			fmt.Sscanf(p.val, "%d", &maxDepth)
		}
	}

	exitCode := 0
	for _, root := range paths {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if maxDepth >= 0 {
				depth := strings.Count(path[len(root):], string(filepath.Separator))
				if depth > maxDepth {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
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
					default:
						match = false
					}
				case "size":
					// Simplified size matching
				case "print", "":
					// always matches
				case "path":
					if path != p.val {
						match = false
					}
				}
			}
			if match {
				fmt.Println(path)
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

		re, err := regexp.Compile(searchStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: sed: invalid pattern: %s\n", err)
			return 1
		}

		result := ""
		if global {
			result = re.ReplaceAllString(string(data), replaceStr)
		} else {
			result = re.ReplaceAllStringFunc(string(data), func(match string) string {
				if result == "" {
					return re.ReplaceAllString(match, replaceStr)
				}
				return match
			})
			if result == "" {
				result = string(data)
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

	data1, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: diff: %s: %v\n", args[1], err)
		return 1
	}
	data2, err := os.ReadFile(args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: diff: %s: %v\n", args[2], err)
		return 1
	}

	lines1 := strings.Split(strings.TrimRight(string(data1), "\n"), "\n")
	lines2 := strings.Split(strings.TrimRight(string(data2), "\n"), "\n")

	// Simple diff using longest common subsequence
	diff := buildDiff(lines1, lines2)
	for _, d := range diff {
		switch d.op {
		case '=':
			// context line
		case '-':
			fmt.Printf("< %s\n", d.text)
		case '+':
			fmt.Printf("> %s\n", d.text)
		}
	}

	if len(diff) > 0 {
		return 1
	}
	return 0
}

type diffLine struct {
	op   byte // '=', '-', '+'
	text string
}

func buildDiff(a, b []string) []diffLine {
	// LCS-based diff
	m, n := len(a), len(b)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
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
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
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

func patchMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: patch: not fully implemented")
	return 1
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
