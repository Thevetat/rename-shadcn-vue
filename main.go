package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var globalRenames = make(map[string]string)

func toKebabCase(s string) string {
	s = strings.ReplaceAll(s, "UI", "Ui")

	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteRune('-')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

func isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}

	if !unicode.IsUpper(rune(s[0])) {
		return false
	}

	hasMoreUpper := false
	for _, r := range s[1:] {
		if unicode.IsUpper(r) {
			hasMoreUpper = true
			break
		}
	}
	return hasMoreUpper
}

func findPascalCaseImports(content string) []string {
	importRegex := regexp.MustCompile(`(?:import|from ['"]\./|from ['"]@/components/ui/[^'"]+/)([A-Z][a-zA-Z]+)`)
	matches := importRegex.FindAllStringSubmatch(content, -1)

	found := make(map[string]bool)
	var results []string

	for _, match := range matches {
		if len(match) > 1 && isPascalCase(match[1]) {
			componentName := match[1]
			if !found[componentName] {
				found[componentName] = true
				results = append(results, componentName)
			}
		}
	}

	return results
}

func findComponentsDir() (string, error) {
	commonPaths := []string{
		"app/components",
		"components",
		"src/components",
		"src/app/components",
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting current directory: %v", err)
	}

	for _, basePath := range commonPaths {
		path := filepath.Join(cwd, basePath, "ui")
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			fmt.Printf("Found components directory: %s\n", path)
			return path, nil
		}
	}

	for _, path := range commonPaths {
		fullPath := filepath.Join(cwd, path)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			fmt.Printf("Found components directory: %s\n", fullPath)
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("could not find components directory in common locations. Please provide the path as an argument")
}

func buildRenameMap(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range entries {
		if !f.IsDir() && (strings.HasSuffix(f.Name(), ".vue") || strings.HasSuffix(f.Name(), ".ts")) {
			filePath := filepath.Join(dir, f.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			pascalImports := findPascalCaseImports(string(content))
			for _, name := range pascalImports {
				if _, exists := globalRenames[name]; !exists {
					newName := toKebabCase(name)
					globalRenames[name] = newName
					fmt.Printf("Found PascalCase import to rename: %s -> %s in %s\n", name, newName, filePath)
				}
			}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())
			if err := buildRenameMap(subdir); err != nil {
				return err
			}
		}
	}

	return nil
}

func updateFileContent(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	newContent := string(content)
	originalContent := newContent

	for oldName, newName := range globalRenames {
		importPatterns := []struct {
			old string
			new string
		}{
			{fmt.Sprintf("from '@/components/ui/sheet/%s.vue'", oldName), fmt.Sprintf("from '@/components/ui/sheet/%s.vue'", newName)},
			{fmt.Sprintf("from '@/components/ui/sheet/%s'", oldName), fmt.Sprintf("from '@/components/ui/sheet/%s'", newName)},
			{fmt.Sprintf("from '@/components/ui/%s/%s.vue'", strings.ToLower(oldName), oldName), fmt.Sprintf("from '@/components/ui/%s/%s.vue'", strings.ToLower(newName), newName)},
			{fmt.Sprintf("from '@/components/ui/%s'", oldName), fmt.Sprintf("from '@/components/ui/%s'", newName)},

			{fmt.Sprintf("import %s from '@/components/ui/sheet/%s.vue'", oldName, oldName), fmt.Sprintf("import %s from '@/components/ui/sheet/%s.vue'", oldName, newName)},
			{fmt.Sprintf("import %s from '@/components/ui/%s/%s.vue'", oldName, strings.ToLower(oldName), oldName), fmt.Sprintf("import %s from '@/components/ui/%s/%s.vue'", oldName, strings.ToLower(newName), newName)},
			{fmt.Sprintf("import %s from './sheet/%s.vue'", oldName, oldName), fmt.Sprintf("import %s from './sheet/%s.vue'", oldName, newName)},

			{fmt.Sprintf("from './%s.vue'", oldName), fmt.Sprintf("from './%s.vue'", newName)},
			{fmt.Sprintf("from './%s'", oldName), fmt.Sprintf("from './%s'", newName)},
			{fmt.Sprintf("from '../%s.vue'", oldName), fmt.Sprintf("from '../%s.vue'", newName)},
			{fmt.Sprintf("from '../%s'", oldName), fmt.Sprintf("from '../%s'", newName)},

			{fmt.Sprintf("import %s from './%s.vue'", oldName, oldName), fmt.Sprintf("import %s from './%s.vue'", oldName, newName)},

			{fmt.Sprintf("/%s/%s.vue'", oldName, oldName), fmt.Sprintf("/%s/%s.vue'", newName, newName)},
			{fmt.Sprintf("/%s/%s'", oldName, oldName), fmt.Sprintf("/%s/%s'", newName, newName)},
		}

		for _, pattern := range importPatterns {
			if strings.Contains(newContent, pattern.old) {
				fmt.Printf("Found import to update in %s: %s -> %s\n", filePath, pattern.old, pattern.new)
				newContent = strings.ReplaceAll(newContent, pattern.old, pattern.new)
			}
		}
	}

	if newContent != originalContent {
		fmt.Printf("Updated imports in: %s\n", filePath)
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}
	return nil
}

func processFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range entries {
		if !f.IsDir() {
			filePath := filepath.Join(dir, f.Name())
			ext := filepath.Ext(f.Name())
			if ext == ".vue" || ext == ".ts" {
				if err := updateFileContent(filePath); err != nil {
					return err
				}
			}
		}
	}

	for oldName, newName := range globalRenames {
		oldPath := filepath.Join(dir, oldName+".vue")
		if _, err := os.Stat(oldPath); err == nil {
			newPath := filepath.Join(dir, newName+".vue")
			if oldPath != newPath {
				if err := os.Rename(oldPath, newPath); err != nil {
					return err
				}
				fmt.Printf("Renamed: %s -> %s\n", oldPath, newPath)
			}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())
			if err := processFiles(subdir); err != nil {
				return err
			}
		}
	}

	return nil
}

func confirmChanges() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nDo you want to proceed with these changes? (y/n): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func main() {
	var dir string
	var err error

	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else {
		dir, err = findComponentsDir()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println("Usage: rename_shadcn [components_directory]")
			os.Exit(1)
		}
	}

	if err := buildRenameMap(dir); err != nil {
		fmt.Printf("Error building rename map: %v\n", err)
		os.Exit(1)
	}

	if len(globalRenames) == 0 {
		fmt.Println("No PascalCase imports found to rename.")
		os.Exit(0)
	}

	fmt.Println("\nProposed changes:")
	fmt.Println("=================")
	for old, new := range globalRenames {
		fmt.Printf("%s -> %s\n", old, new)
	}
	fmt.Println("\nThis will update all imports in .vue and .ts files to use the new kebab-case names.")

	if !confirmChanges() {
		fmt.Println("Operation cancelled.")
		os.Exit(0)
	}

	fmt.Println("\nProceeding with changes...")
	if err := processFiles(dir); err != nil {
		fmt.Printf("Error processing files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nAll changes completed successfully!")
}
