package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nhancdt2602/cachewrap/tool/instrument"

	"github.com/dave/dst/decorator"
)

const (
	SubcommandRemix = "remix"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case SubcommandRemix:
		// Called by -toolexec: cachewrap remix <compiler> <args...>
		if err := handleRemix(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error in remix: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`CacheWrap - Compile-time automatic caching for Go

Usage:
  # Build with automatic cache instrumentation
  go build -toolexec="cachewrap remix" -a main.go

  # Install with instrumentation
  go install -toolexec="cachewrap remix" -a

The 'remix' subcommand is used internally by -toolexec to intercept compilation.
`)
}

// handleRemix is called by Go's -toolexec flag
// It intercepts the compiler invocation and instruments source files on the flight
func handleRemix(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no compiler command provided")
	}

	compiler := args[0]
	compilerArgs := args[1:]

	if !strings.Contains(compiler, "compile") {
		return runCommand(compiler, compilerArgs)
	}

	// Find Go source files in compiler arguments
	goFiles := []string{}
	otherArgs := []string{}

	for _, arg := range compilerArgs {
		if strings.HasSuffix(arg, ".go") && !strings.HasPrefix(filepath.Base(arg), ".") {
			goFiles = append(goFiles, arg)
		} else {
			otherArgs = append(otherArgs, arg)
		}
	}

	// If no Go files, just pass through
	if len(goFiles) == 0 {
		return runCommand(compiler, compilerArgs)
	}

	// Instrument each Go file
	instrumentedFiles := make([]string, 0, len(goFiles))
	tempDir := os.TempDir()

	for _, goFile := range goFiles {
		rules, dstFile, dec, err := instrument.ParseCachewrapAnnotations(goFile)
		if err != nil {
			// Fallback to original file if parsing fails
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse %s: %v\n", goFile, err)
			instrumentedFiles = append(instrumentedFiles, goFile)
			continue
		}

		if len(rules) == 0 {
			instrumentedFiles = append(instrumentedFiles, goFile)
			continue
		}

		if err := instrument.InjectCaching(dstFile, dec, rules); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to inject caching for %s: %v\n", goFile, err)
			instrumentedFiles = append(instrumentedFiles, goFile)
			continue
		}

		tempFile := filepath.Join(tempDir, "cachewrap_"+filepath.Base(goFile))
		f, err := os.Create(tempFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create temp file for %s: %v\n", goFile, err)
			instrumentedFiles = append(instrumentedFiles, goFile)
			continue
		}

		restorer := decorator.NewRestorer()
		if err := restorer.Fprint(f, dstFile); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "Warning: Failed to write instrumented %s: %v\n", goFile, err)
			instrumentedFiles = append(instrumentedFiles, goFile)
			continue
		}
		f.Close()

		err = instrument.EnableLineDirective(tempFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not enable line directive in %s: %v\n", tempFile, err)
			instrumentedFiles = append(instrumentedFiles, goFile)
			continue
		}

		instrumentedFiles = append(instrumentedFiles, tempFile)
	}

	newArgs := append(otherArgs, instrumentedFiles...)

	// Rewrite original arguments with new arguments containing instrumented files
	return runCommand(compiler, newArgs)
}

func runCommand(command string, args []string) error {
	cmd := newCommand(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func newCommand(name string, args ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		if !hasExeExtension(name) {
			name = name + ".exe"
		}
	}
	return exec.Command(name, args...)
}

func hasExeExtension(name string) bool {
	return len(name) >= 4 && name[len(name)-4:] == ".exe"
}
