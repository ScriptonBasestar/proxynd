package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"bytes"
)

// This tool automatically fixes deprecated io/ioutil usage
func main() {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files and vendor directory
		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "vendor/") {
			return nil
		}

		// Skip this file itself
		if strings.Contains(path, "auto_fix_ioutil.go") {
			return nil
		}

		return processFile(path)
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func processFile(filename string) error {
	// Read the file
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return err
	}

	modified := false

	// Check and fix imports
	for _, imp := range file.Imports {
		if imp.Path.Value == `"io/ioutil"` {
			// Check what ioutil functions are used
			hasReadAll := false
			needsOS := false
			
			ast.Inspect(file, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "ioutil" {
						switch sel.Sel.Name {
						case "ReadAll":
							hasReadAll = true
						case "ReadFile", "WriteFile", "ReadDir", "TempFile", "TempDir":
							needsOS = true
						}
					}
				}
				return true
			})

			if hasReadAll && !needsOS {
				imp.Path.Value = `"io"`
				modified = true
			} else if needsOS && !hasReadAll {
				imp.Path.Value = `"os"`
				modified = true
			} else if needsOS && hasReadAll {
				// Need both io and os
				imp.Path.Value = `"os"`
				// Add io import if not present
				hasIO := false
				for _, i := range file.Imports {
					if i.Path.Value == `"io"` {
						hasIO = true
						break
					}
				}
				if !hasIO {
					// This is complex, just mark for manual review
					fmt.Printf("WARNING: %s needs both 'io' and 'os' imports. Please review manually.\n", filename)
				}
				modified = true
			}
		}
	}

	// Replace ioutil function calls
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "ioutil" {
				switch sel.Sel.Name {
				case "ReadFile", "WriteFile", "ReadDir", "TempFile", "TempDir":
					ident.Name = "os"
					modified = true
				case "ReadAll":
					ident.Name = "io"
					modified = true
				case "NopCloser":
					ident.Name = "io"
					modified = true
				case "Discard":
					ident.Name = "io"
					modified = true
				}
			}
		}
		return true
	})

	if !modified {
		return nil
	}

	// Format and write back
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Printf("Fixed: %s\n", filename)
	return nil
}