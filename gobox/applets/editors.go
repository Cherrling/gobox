package applets

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func init() {
	Register("vi", AppletFunc(viMain))
	Register("ed", AppletFunc(edMain))
	Register("awk", AppletFunc(awkMain))
}

func viMain(args []string) int {
	path := ""
	if len(args) > 1 {
		path = args[1]
	}

	// Check if we're in a terminal
	info, _ := os.Stdout.Stat()
	if info.Mode()&os.ModeCharDevice == 0 {
		fmt.Fprintln(os.Stderr, "gobox: vi: not a terminal")
		return 1
	}

	if path == "" {
		fmt.Fprintln(os.Stderr, "gobox: vi: no file specified")
		return 1
	}

	// Simple line-based editor
	data, _ := os.ReadFile(path)
	lines := strings.Split(string(data), "\n")
	if len(data) == 0 {
		lines = []string{""}
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("%d lines, %d characters\n", len(lines), len(data))

	for {
		fmt.Print(":")
		if !scanner.Scan() {
			break
		}
		cmd := strings.TrimSpace(scanner.Text())

		switch {
		case cmd == "q" || cmd == "q!":
			break
		case cmd == "w" || cmd == "wq":
			os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
			if cmd == "wq" {
				return 0
			}
		case cmd == "w!":
			os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		case strings.HasPrefix(cmd, "w "):
			parts := strings.Fields(cmd)
			if len(parts) >= 2 {
				os.WriteFile(parts[1], []byte(strings.Join(lines, "\n")), 0644)
			}
		case strings.HasPrefix(cmd, "s/"):
			// s/old/new/
			parts := strings.SplitN(cmd[2:], "/", 3)
			if len(parts) >= 2 {
				oldStr := parts[0]
				newStr := parts[1]
				for i, line := range lines {
					lines[i] = strings.Replace(line, oldStr, newStr, 1)
				}
			}
		case cmd == "p" || cmd == "%p":
			for _, line := range lines {
				fmt.Println(line)
			}
		case strings.HasPrefix(cmd, "p "):
			// print range
		case cmd == "n" || cmd == "n!":
			// next file (not supported)
		default:
			fmt.Println("Unknown command:", cmd)
		}

		if cmd == "q" || cmd == "q!" {
			break
		}
	}
	return 0
}

func edMain(args []string) int {
	// ed is the standard text editor
	path := ""
	if len(args) > 1 {
		path = args[1]
	}

	var lines []string
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			lines = strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			fmt.Printf("%d\n", len(lines))
		} else {
			fmt.Printf("%s: No such file or directory\n", path)
			lines = []string{}
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := strings.TrimSpace(scanner.Text())
		if cmd == "" {
			continue
		}
		if cmd == "q" || cmd == "Q" {
			break
		}
		if cmd == "w" && path != "" {
			os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
			fmt.Printf("%d\n", len(strings.Join(lines, "\n"))+1)
			continue
		}
		if cmd == "wq" && path != "" {
			os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
			break
		}
		if cmd == "p" || cmd == ",p" || cmd == "%p" {
			for _, l := range lines {
				fmt.Println(l)
			}
			continue
		}
		if cmd == "n" || cmd == ",n" || cmd == "%n" {
			for i, l := range lines {
				fmt.Printf("%d\t%s\n", i+1, l)
			}
			continue
		}
		if cmd == "=" {
			fmt.Println(len(lines))
			continue
		}
		if cmd == "h" {
			fmt.Println("?")
			continue
		}
		fmt.Println("?")
	}
	return 0
}

func awkMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: awk: missing program")
		return 1
	}

	program := args[1]
	_ = program

	// Simple awk: just print lines
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		_ = fields

		if strings.HasPrefix(program, "{print $") {
			// Extract field number
			var n int
			if _, err := fmt.Sscanf(program, "{print $%d}", &n); err == nil {
				if n > 0 && n <= len(fields) {
					fmt.Println(fields[n-1])
				}
				continue
			}
		}
		if program == "{print}" || program == "{print $0}" {
			fmt.Println(line)
			continue
		}
		if program == "{print NF}" {
			fmt.Println(len(fields))
			continue
		}
		if program == "{print NR}" {
			// Just print line number
			continue
		}
		fmt.Println(line)
	}
	return 0
}

