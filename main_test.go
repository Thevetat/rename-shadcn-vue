package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single word", "Button", "button"},
		{"two words", "ButtonGroup", "button-group"},
		{"UI prefix", "UIButton", "ui-button"},
		{"numbers", "Button2Group", "button2-group"},
		{"multiple uppercase", "HTMLInput", "html-input"},
		{"already kebab", "button-group", "button-group"},
		{"complex name", "DialogContentPanel", "dialog-content-panel"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := toKebabCase(tc.input)
			if result != tc.expected {
				t.Errorf("toKebabCase(%q) = %q; want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestIsPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"component name", "Button", true},
		{"component with suffix", "ButtonGroup", true},
		{"props type", "ButtonProps", false},
		{"emits type", "ButtonEmits", false},
		{"context type", "ButtonContext", false},
		{"utility type", "HTMLAttributes", false},
		{"constant", "SOURCE", false},
		{"icon name", "ChevronRight", false},
		{"provider", "ButtonProvider", false},
		{"portal", "ButtonPortal", false},
		{"multi word component", "AccordionTrigger", true},
		{"non-component pascal", "MyClass", false},
		{"kebab case", "button-group", false},
		{"camel case", "buttonGroup", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isPascalCase(tc.input)
			if result != tc.expected {
				t.Errorf("isPascalCase(%q) = %v; want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFindPascalCaseImports(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "no imports",
			content: `const x = 1
export default {}`,
			expected: nil,
		},
		{
			name: "multiple imports same component",
			content: `import Button from './Button.vue'
import { Button } from '@/components/ui/Button'`,
			expected: []string{"Button"},
		},
		{
			name: "scoped imports",
			content: `import * as Components from './Button'
import { Button as CustomButton } from './Button'`,
			expected: []string{"Button"},
		},
		{
			name: "multiline imports",
			content: `import {
  Button,
  ButtonGroup
} from '@/components/ui/Button'`,
			expected: []string{"Button", "ButtonGroup"},
		},
		{
			name: "commented imports",
			content: `// import Button from './Button.vue'
/* import Dialog from './Dialog.vue' */`,
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := findPascalCaseImports(tc.content)
			if len(result) != len(tc.expected) {
				t.Errorf("findPascalCaseImports() got %v; want %v", result, tc.expected)
				return
			}
			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("findPascalCaseImports() got %v; want %v", result, tc.expected)
					break
				}
			}
		})
	}
}

type testCase struct {
	name     string
	input    string
	expected string
	renames  map[string]string
}

func TestUpdateFileContent(t *testing.T) {
	tests := []testCase{
		{
			name: "basic import",
			input: `import Accordion from './Accordion.vue'
import Button from './Button.vue'`,
			expected: `import Accordion from './accordion.vue'
import Button from './button.vue'`,
			renames: map[string]string{
				"Accordion": "accordion",
				"Button":    "button",
			},
		},
		{
			name: "barrel exports",
			input: `export { default as Accordion } from './Accordion.vue'
export { default as AccordionContent } from './AccordionContent.vue'
export { default as AccordionItem } from './AccordionItem.vue'
export { default as AccordionTrigger } from './AccordionTrigger.vue'`,
			expected: `export { default as Accordion } from './accordion.vue'
export { default as AccordionContent } from './accordion-content.vue'
export { default as AccordionItem } from './accordion-item.vue'
export { default as AccordionTrigger } from './accordion-trigger.vue'`,
			renames: map[string]string{
				"Accordion":        "accordion",
				"AccordionContent": "accordion-content",
				"AccordionItem":    "accordion-item",
				"AccordionTrigger": "accordion-trigger",
			},
		},
		{
			name: "alias imports",
			input: `import Button from '@/components/ui/Button.vue'
import Input from '~/components/ui/Input.vue'
import { Dialog } from '@/components/ui/Dialog'`,
			expected: `import Button from '@/components/ui/button.vue'
import Input from '~/components/ui/input.vue'
import { Dialog } from '@/components/ui/dialog'`,
			renames: map[string]string{
				"Button": "button",
				"Input":  "input",
				"Dialog": "dialog",
			},
		},
		{
			name: "relative path imports",
			input: `import Card from '../Card.vue'
import { Avatar } from '../../Avatar'
import Select from './Select/Select.vue'`,
			expected: `import Card from '../card.vue'
import { Avatar } from '../../avatar'
import Select from './select/select.vue'`,
			renames: map[string]string{
				"Card":   "card",
				"Avatar": "avatar",
				"Select": "select",
			},
		},
		{
			name: "mixed content",
			input: `import { Button } from '@/components/ui/Button'

import ButtonGroup from './ButtonGroup.vue'
export { default as ButtonIcon } from './ButtonIcon.vue'

const template = '<Button>Click me</Button>'`,
			expected: `import { Button } from '@/components/ui/button'

import ButtonGroup from './button-group.vue'
export { default as ButtonIcon } from './button-icon.vue'

const template = '<Button>Click me</Button>'`,
			renames: map[string]string{
				"Button":      "button",
				"ButtonGroup": "button-group",
				"ButtonIcon":  "button-icon",
			},
		},
		{
			name: "directory paths",
			input: `import Dialog from '@/components/ui/Dialog/Dialog.vue'
import { DialogContent } from '@/components/ui/Dialog/DialogContent'`,
			expected: `import Dialog from '@/components/ui/dialog/dialog.vue'
import { DialogContent } from '@/components/ui/dialog/dialog-content'`,
			renames: map[string]string{
				"Dialog":        "dialog",
				"DialogContent": "dialog-content",
			},
		},
		{
			name: "no extension imports",
			input: `import { Tabs } from '@/components/ui/Tabs'
import { TabsList } from '@/components/ui/TabsList'`,
			expected: `import { Tabs } from '@/components/ui/tabs'
import { TabsList } from '@/components/ui/tabs-list'`,
			renames: map[string]string{
				"Tabs":     "tabs",
				"TabsList": "tabs-list",
			},
		},
		{
			name:     "destructured imports",
			input:    `import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/Popover'`,
			expected: `import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'`,
			renames: map[string]string{
				"Popover":        "popover",
				"PopoverContent": "popover-content",
				"PopoverTrigger": "popover-trigger",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			tmpDir, err := os.MkdirTemp("", "rename_test_*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			tmpFile := filepath.Join(tmpDir, "test.vue")
			if err := os.WriteFile(tmpFile, []byte(tc.input), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			globalRenames = tc.renames

			if err := updateFileContent(tmpFile); err != nil {
				t.Fatalf("updateFileContent failed: %v", err)
			}

			result, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to read result file: %v", err)
			}

			if string(result) != tc.expected {
				t.Errorf("\nExpected:\n%s\n\nGot:\n%s", tc.expected, string(result))
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rename_test_integration_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	componentsDir := filepath.Join(tmpDir, "components", "ui")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		t.Fatalf("Failed to create components directory: %v", err)
	}

	files := map[string]string{
		"Button.vue": `import Card from '../Card.vue'
export default {}`,
		"Card.vue": `import Button from './Button.vue'
export default {}`,
		"Dialog/Dialog.vue": `import DialogContent from './DialogContent.vue'
export default {}`,
		"Dialog/DialogContent.vue": `import Dialog from './Dialog.vue'
export default {}`,
	}

	for path, content := range files {
		fullPath := filepath.Join(componentsDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	if err := buildRenameMap(componentsDir); err != nil {
		t.Fatalf("buildRenameMap failed: %v", err)
	}

	if err := processFiles(componentsDir); err != nil {
		t.Fatalf("processFiles failed: %v", err)
	}

	expectedFiles := []string{
		"button.vue",
		"card.vue",
		"dialog/dialog.vue",
		"dialog/dialog-content.vue",
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(filepath.Join(componentsDir, file)); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", file)
		}
	}
}
