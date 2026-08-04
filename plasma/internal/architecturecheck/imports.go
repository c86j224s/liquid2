package architecturecheck

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type importEdge struct {
	file       string
	importPath string
}

func scanGoImports(moduleRoot string) ([]importEdge, error) {
	edges := make([]importEdge, 0)
	err := filepath.WalkDir(moduleRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor":
				if path != moduleRoot {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		relative, err := filepath.Rel(moduleRoot, path)
		if err != nil {
			return fmt.Errorf("relativize %s: %w", path, err)
		}
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return fmt.Errorf("unquote import in %s: %w", path, err)
			}
			edges = append(edges, importEdge{
				file:       filepath.ToSlash(relative),
				importPath: importPath,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].file == edges[j].file {
			return edges[i].importPath < edges[j].importPath
		}
		return edges[i].file < edges[j].file
	})
	return edges, nil
}

func readBaseline(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Strings(lines)
	return lines, nil
}

func writeBaseline(path string, lines []string) error {
	sorted := append([]string(nil), lines...)
	sort.Strings(sorted)
	content := "# Known package-boundary debt.\n" +
		"# Format: rule<TAB>file<TAB>import path\n" +
		"# Remove entries as refactors eliminate the corresponding import.\n"
	if len(sorted) > 0 {
		content += strings.Join(sorted, "\n") + "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func compareLines(expected, actual []string) (missing, added []string) {
	expectedSet := make(map[string]struct{}, len(expected))
	actualSet := make(map[string]struct{}, len(actual))
	for _, line := range expected {
		expectedSet[line] = struct{}{}
	}
	for _, line := range actual {
		actualSet[line] = struct{}{}
	}
	for _, line := range expected {
		if _, ok := actualSet[line]; !ok {
			missing = append(missing, line)
		}
	}
	for _, line := range actual {
		if _, ok := expectedSet[line]; !ok {
			added = append(added, line)
		}
	}
	return missing, added
}
