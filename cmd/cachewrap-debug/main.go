package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dave/dst/decorator"
	"github.com/nhancdt2602/cachewrap/tool/instrument"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <go-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s example/repo.go\n", os.Args[0])
		os.Exit(1)
	}

	filePath := os.Args[1]

	fmt.Println("=== CacheWrap Instrumentation Debug ===")
	fmt.Printf("Processing: %s\n", filePath)

	// Parse file for cachewrap annotations
	cacheRules, dstFile, dec, err := instrument.ParseCachewrapAnnotations(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d cache annotations:\n", len(cacheRules))
	for i, rule := range cacheRules {
		receiver := ""
		if rule.ReceiverType != "" {
			receiver = fmt.Sprintf("(%s).", rule.ReceiverType)
		}

		// Determine annotation type
		annotationType := "cachewrap"
		if rule.IsEvict {
			annotationType = "cacheevict"
		} else if rule.IsClear {
			annotationType = "cacheclear"
		}

		// Show full details
		fmt.Printf("  %d. [%s] %s%s\n", i+1, annotationType, receiver, rule.FuncName)

		if rule.KeyPrefix != "" {
			fmt.Printf("	- prefix: %s\n", rule.KeyPrefix)
		}
		if rule.CacheInstance != "" {
			fmt.Printf("	- cache: %s\n", rule.CacheInstance)
		}
		if len(rule.ParamNames) > 0 {
			fmt.Printf("	- params: %v\n", rule.ParamNames)
		}
	}

	if len(cacheRules) == 0 {
		fmt.Println("No cachewrap annotations found")
		return
	}

	// Inject caching logic
	fmt.Println("Injecting caching logic...")
	err = instrument.InjectCaching(dstFile, dec, cacheRules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error injecting caching: %v\n", err)
		os.Exit(1)
	}

	// Print instrumented code
	fmt.Println("=== INSTRUMENTED CODE ===")
	restorer := decorator.NewRestorer()
	err = restorer.Fprint(os.Stdout, dstFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error printing code: %v\n", err)
		os.Exit(1)
	}

	debugDir := ".cache-build/instrument"
	os.MkdirAll(debugDir, 0755)

	outputFile := filepath.Join(debugDir, filepath.Base(filePath))
	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not save to %s: %v\n", outputFile, err)
		os.Exit(1)
	}

	defer f.Close()
	// Create new restorer for file output to avoid node duplication
	fileRestorer := decorator.NewRestorer()
	if err := fileRestorer.Fprint(f, dstFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not write to %s: %v\n", outputFile, err)
		os.Exit(1)
	}

	err = instrument.EnableLineDirective(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not enable line directive in %s: %v\n", outputFile, err)
		os.Exit(1)
	}
	fmt.Println("=== SAVED TO ===")
	fmt.Println(outputFile)

}
