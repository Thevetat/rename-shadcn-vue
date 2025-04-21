# Rename Shadcn Vue

A utility for automatically renaming Shadcn Vue components from PascalCase to kebab-case naming conventions.

## Installation

```bash
git clone https://github.com/yourusername/rename-shadcn-vue.git
cd rename-shadcn-vue
go build
```

## Usage

Run the tool without arguments while in the top level of your project to automatically locate your components directory:

```bash
./rename-shadcn-vue
```

Or specify a components directory:

```bash
./rename-shadcn-vue path/to/components
```

## How It Works

1. Scans your project for Shadcn Vue components with PascalCase naming
2. Converts these names to kebab-case
3. Updates all import statements in .vue and .ts files
4. Renames the component files themselves

The tool will display all proposed changes and ask for confirmation before proceeding.

## Features

- Automatic components directory detection
- Converts PascalCase to kebab-case (e.g., `AlertDialog` → `alert-dialog`)
- Updates import paths in all .vue and .ts files
- Interactive confirmation before making changes 

## ⚠️ Disclaimer

- **Backup your project**: Always make sure your project is committed to version control or properly backed up before running this tool. This will allow you to revert changes if something unexpected happens. 