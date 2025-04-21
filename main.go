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
	var prevIsUpper bool
	for i, r := range s {
		isUpper := unicode.IsUpper(r)

		if i > 0 && isUpper && (!prevIsUpper || (i+1 < len(s) && unicode.IsLower(rune(s[i+1])))) {
			result.WriteRune('-')
		}

		result.WriteRune(unicode.ToLower(r))
		prevIsUpper = isUpper
	}
	return result.String()
}

func isPascalCase(s string) bool {
	if strings.HasSuffix(s, "Props") || strings.HasSuffix(s, "Emits") || strings.HasSuffix(s, "Context") {
		return false
	}

	skipWords := []string{"HTML", "Ref", "VModel", "Component", "Primitive", "Variants", "Omit",
		"NAME", "AGE", "ICON", "WIDTH", "MOBILE", "SHORTCUT", "SOURCE", "Provider", "Portal"}
	for _, word := range skipWords {
		if strings.Contains(s, word) {
			return false
		}
	}

	componentPrefixes := []string{
		"Sidebar",
		"Accordion",
		"Alert",
		"AlertDialog",
		"AspectRatio",
		"Avatar",
		"Badge",
		"Breadcrumb",
		"Button",
		"Calendar",
		"Card",
		"Carousel",
		"Checkbox",
		"Collapsible",
		"Combobox",
		"Command",
		"ContextMenu",
		"DataTable",
		"DatePicker",
		"Dialog",
		"Drawer",
		"DropdownMenu",
		"Form",
		"HoverCard",
		"Input",
		"Label",
		"Menubar",
		"NavigationMenu",
		"NumberField",
		"Pagination",
		"PinInput",
		"Popover",
		"Progress",
		"RadioGroup",
		"RangeCalendar",
		"Resizable",
		"ScrollArea",
		"Select",
		"Separator",
		"Sheet",
		"Skeleton",
		"Slider",
		"Sonner",
		"Stepper",
		"Switch",
		"Table",
		"Tabs",
		"TagsInput",
		"Textarea",
		"Toast",
		"Toggle",
		"ToggleGroup",
		"Tooltip",
	}

	for _, prefix := range componentPrefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}

	return false
}

func findPascalCaseImports(content string) []string {
	found := make(map[string]bool)
	var results []string

	singleLineCommentRegex := regexp.MustCompile(`//.*$`)
	lines := strings.Split(content, "\n")
	var cleanedLines []string
	for _, line := range lines {
		cleanedLine := singleLineCommentRegex.ReplaceAllString(line, "")
		if strings.TrimSpace(cleanedLine) != "" {
			cleanedLines = append(cleanedLines, cleanedLine)
		}
	}
	cleanContent := strings.Join(cleanedLines, "\n")

	multiLineCommentRegex := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	cleanContent = multiLineCommentRegex.ReplaceAllString(cleanContent, "")

	patterns := []string{
		`import\s+([A-Z][a-zA-Z0-9]+)(?:\s*,\s*([A-Z][a-zA-Z0-9]+))*\s+from`,
		`import\s*{\s*([A-Z][a-zA-Z0-9]+(?:\s*,\s*[A-Z][a-zA-Z0-9]+)*)\s*}\s*from`,
		`from\s+['"].*?/([A-Z][a-zA-Z0-9]+)\.vue['"]`,
		`from\s+['"].*?/([A-Z][a-zA-Z0-9]+)['"]`,
		`export\s*{\s*default\s+as\s+([A-Z][a-zA-Z0-9]+)\s*}\s*from\s*['"]`,
		`export\s*{\s*([A-Z][a-zA-Z0-9]+(?:\s*,\s*[A-Z][a-zA-Z0-9]+)*)\s*}\s*from\s*['"]`,
		`import\s*{\s*([A-Z][a-zA-Z0-9]+(?:\s*,\s*[A-Z][a-zA-Z0-9]+)*)\s*}\s*from\s*['"].*?/[A-Z][a-zA-Z]+['"]`,
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(cleanContent, -1)
		for _, match := range matches {
			for i := 1; i < len(match); i++ {
				if match[i] == "" {
					continue
				}
				components := strings.Split(match[i], ",")
				for _, component := range components {
					component = strings.TrimSpace(component)
					if component != "" && isPascalCase(component) {
						if !found[component] {
							found[component] = true
							results = append(results, component)
						}
					}
				}
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

		stringPatterns := []struct {
			old string
			new string
		}{

			{fmt.Sprintf("export { default as %s } from './%s.vue'", oldName, oldName), fmt.Sprintf("export { default as %s } from './%s.vue'", oldName, newName)},

			{fmt.Sprintf("from '@/components/ui/%s.vue'", oldName), fmt.Sprintf("from '@/components/ui/%s.vue'", newName)},
			{fmt.Sprintf("from '@/components/ui/%s'", oldName), fmt.Sprintf("from '@/components/ui/%s'", newName)},
			{fmt.Sprintf("from '~/components/ui/%s.vue'", oldName), fmt.Sprintf("from '~/components/ui/%s.vue'", newName)},
			{fmt.Sprintf("from '~/components/ui/%s'", oldName), fmt.Sprintf("from '~/components/ui/%s'", newName)},

			{fmt.Sprintf("from './%s.vue'", oldName), fmt.Sprintf("from './%s.vue'", newName)},
			{fmt.Sprintf("from './%s'", oldName), fmt.Sprintf("from './%s'", newName)},
			{fmt.Sprintf("from '../%s.vue'", oldName), fmt.Sprintf("from '../%s.vue'", newName)},
			{fmt.Sprintf("from '../%s'", oldName), fmt.Sprintf("from '../%s'", newName)},
			{fmt.Sprintf("from '../../%s.vue'", oldName), fmt.Sprintf("from '../../%s.vue'", newName)},
			{fmt.Sprintf("from '../../%s'", oldName), fmt.Sprintf("from '../../%s'", newName)},

			{fmt.Sprintf("import %s from '@/components/ui/%s.vue'", oldName, oldName), fmt.Sprintf("import %s from '@/components/ui/%s.vue'", oldName, newName)},
			{fmt.Sprintf("import %s from '~/components/ui/%s.vue'", oldName, oldName), fmt.Sprintf("import %s from '~/components/ui/%s.vue'", oldName, newName)},
			{fmt.Sprintf("import %s from './%s.vue'", oldName, oldName), fmt.Sprintf("import %s from './%s.vue'", oldName, newName)},
			{fmt.Sprintf("import { %s } from '@/components/ui/%s'", oldName, oldName), fmt.Sprintf("import { %s } from '@/components/ui/%s'", oldName, newName)},

			{fmt.Sprintf("/%s/%s.vue'", oldName, oldName), fmt.Sprintf("/%s/%s.vue'", newName, newName)},
			{fmt.Sprintf("/%s/%s'", oldName, oldName), fmt.Sprintf("/%s/%s'", newName, newName)},

			{fmt.Sprintf("from '@/components/ui/%s/%s.vue'", oldName, oldName), fmt.Sprintf("from '@/components/ui/%s/%s.vue'", newName, newName)},
			{fmt.Sprintf("from '@/components/ui/%s/%s'", oldName, oldName), fmt.Sprintf("from '@/components/ui/%s/%s'", newName, newName)},
			{fmt.Sprintf("import %s from '@/components/ui/%s/%s.vue'", oldName, oldName, oldName), fmt.Sprintf("import %s from '@/components/ui/%s/%s.vue'", oldName, newName, newName)},
			{fmt.Sprintf("import { %s } from '@/components/ui/%s/%s'", oldName, oldName, oldName), fmt.Sprintf("import { %s } from '@/components/ui/%s/%s'", oldName, newName, newName)},

			{fmt.Sprintf("from '@/components/ui/%s/%s'", oldName, oldName+"Content"), fmt.Sprintf("from '@/components/ui/%s/%s'", newName, newName+"-content")},
			{fmt.Sprintf("import { %sContent } from '@/components/ui/%s/%s'", oldName, oldName, oldName+"Content"), fmt.Sprintf("import { %sContent } from '@/components/ui/%s/%s'", oldName, newName, newName+"-content")},
		}

		for _, pattern := range stringPatterns {
			if strings.Contains(newContent, pattern.old) {
				fmt.Printf("Found string pattern to update in %s: %s -> %s\n", filePath, pattern.old, pattern.new)
				newContent = strings.ReplaceAll(newContent, pattern.old, pattern.new)
			}
		}

		regexPatterns := []struct {
			old string
			new string
		}{

			{
				fmt.Sprintf(`([@~/]components/ui/)%s(/[^'"]+)`, oldName),
				fmt.Sprintf(`${1}%s${2}`, newName),
			},

			{
				fmt.Sprintf(`(['"][@~/]components/ui/)%s(['"])`, oldName),
				fmt.Sprintf(`${1}%s${2}`, newName),
			},

			{
				fmt.Sprintf(`([@~/]components/ui/%s/)%s`, oldName, oldName),
				fmt.Sprintf(`${1}%s`, newName),
			},

			{
				fmt.Sprintf(`([@~/]components/ui/%s/)%sContent`, oldName, oldName),
				fmt.Sprintf(`${1}%s-content`, newName),
			},
		}

		for _, pattern := range regexPatterns {
			re := regexp.MustCompile(pattern.old)
			if re.MatchString(newContent) {
				fmt.Printf("Found regex pattern to update in %s: %s -> %s\n", filePath, pattern.old, pattern.new)
				newContent = re.ReplaceAllString(newContent, pattern.new)
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
