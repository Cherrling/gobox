package applets

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Applet is the interface that all applets implement.
type Applet interface {
	// Run executes the applet with the given arguments.
	// args[0] is the applet name (argv[0]).
	Run(args []string) int
}

// UsageDesc returns a one-line description of the applet.
type UsageDesc interface {
	Usage() string
}

// AppletFunc adapts a function to the Applet interface.
type AppletFunc func(args []string) int

func (f AppletFunc) Run(args []string) int {
	return f(args)
}

var registry = map[string]Applet{}

// Register adds an applet to the registry under the given name.
func Register(name string, a Applet) {
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("gobox: duplicate applet registration: %q", name))
	}
	registry[name] = a
}

// List returns all registered applet names, sorted.
func List() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// RunByName runs the applet with the given name and args.
// Returns the exit code.
func RunByName(name string, args []string) int {
	a, ok := registry[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "gobox: applet not found: %s\n", name)
		return 1
	}
	return a.Run(args)
}

// printColumns prints a list of strings in aligned columns.
func printColumns(items []string, colWidth int) {
	if len(items) == 0 {
		return
	}
	// Determine terminal width
	cols := 80
	if w, err := os.Stdout.Stat(); err == nil && (w.Mode()&os.ModeCharDevice) != 0 {
		if w, _, err := getTerminalSize(); err == nil && w > 0 {
			cols = w
		}
	}
	nCols := cols / colWidth
	if nCols == 0 {
		nCols = 1
	}
	nRows := (len(items) + nCols - 1) / nCols
	for row := 0; row < nRows; row++ {
		for col := 0; col < nCols; col++ {
			i := col*nRows + row
			if i >= len(items) {
				continue
			}
			fmt.Print(items[i])
			if col < nCols-1 {
				padding := colWidth - len(items[i])
				if padding < 1 {
					padding = 1
				}
				for k := 0; k < padding; k++ {
					fmt.Print(" ")
				}
			}
		}
		fmt.Println()
	}
}

// getTerminalSize returns the terminal width and height.
// It returns 0, 0, error if not a terminal.
func getTerminalSize() (int, int, error) {
	return 0, 0, fmt.Errorf("not supported")
}

// Dispatch determines which applet to run based on argv[0] and the arguments.
func Dispatch(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "gobox: no arguments\n")
		return 1
	}

	appletName := filepath.Base(args[0])

	// "gobox" or "busybox" - list or run sub-applet
	if appletName == "gobox" || appletName == "busybox" {
		if len(args) >= 2 {
			sub := args[1]
			if sub == "--list" || sub == "-l" {
				printColumns(List(), 8)
				return 0
			}
			if !strings.HasPrefix(sub, "-") {
				return RunByName(sub, append([]string{sub}, args[2:]...))
			}
		}
		// No subcommand or flag: show help
		fmt.Println("gobox: multi-call binary")
		fmt.Println("Usage: gobox [APPLET] [ARGS]...")
		fmt.Println()
		fmt.Println("Available applets:")
		printColumns(List(), 20)
		return 0
	}

	// Try direct applet name match
	if a, ok := registry[appletName]; ok {
		return a.Run(args)
	}

	// Handle symlinked names with hyphens (e.g., ether-wake)
	// busybox normalizes hyphens in applet names
	normalized := strings.ReplaceAll(appletName, "-", "_")
	if a, ok := registry[normalized]; ok {
		return a.Run(append([]string{normalized}, args[1:]...))
	}

	fmt.Fprintf(os.Stderr, "gobox: applet not found: %s\n", appletName)
	return 1
}
