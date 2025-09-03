package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sblinch/kdl-go"
	"github.com/sblinch/kdl-go/document"
)

// Minimal unit tests for core functions
func TestConvertValue(t *testing.T) {
	tests := []struct {
		name     string
		value    *document.Value
		expected interface{}
	}{
		{
			name:     "string value",
			value:    &document.Value{Value: "test"},
			expected: "test",
		},
		{
			name:     "int value",
			value:    &document.Value{Value: int64(42)},
			expected: int64(42),
		},
		{
			name:     "nil value",
			value:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertValue(tt.value)
			if result != tt.expected {
				t.Errorf("convertValue() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// E2E tests using the compiled binary
func TestBasicConversion(t *testing.T) {
	// Check if binary exists before running E2E tests
	if err := checkBinaryExists(); err != nil {
		t.Skipf("Skipping E2E test: %v", err)
	}

	tests := []struct {
		name         string
		kdlContent   string
		expectedJSON string
	}{
		{
			name: "simple scene",
			kdlContent: `scene "TestScene" {
    node "Button" x=100 y=100
}`,
			expectedJSON: `{
  "scene": {
    "arg1": "TestScene",
    "node": {
      "arg1": "Button",
      "x": 100,
      "y": 100
    }
  }
}`,
		},
		{
			name: "various value types",
			kdlContent: `config {
    name "Test"
    version 1.0
    enabled true
    count 42
}`,
			expectedJSON: `{
  "config": {
    "count": 42,
    "enabled": true,
    "name": "Test",
    "version": 1.0
  }
}`,
		},
		{
			name: "complex nested structure",
			kdlContent: `scene "SimpleScene" {
    node "Button" x=100 y=100 width=200 height=50 {
        component "Button" text="Click me"
    }
}`,
			expectedJSON: `{
  "scene": {
    "arg1": "SimpleScene",
    "node": {
      "arg1": "Button",
      "height": 50,
      "width": 200,
      "x": 100,
      "y": 100,
      "component": {
        "arg1": "Button",
        "text": "Click me"
      }
    }
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary KDL file
			tmpDir := t.TempDir()
			kdlFile := filepath.Join(tmpDir, "test.kdl")
			if err := os.WriteFile(kdlFile, []byte(tt.kdlContent), 0644); err != nil {
				t.Fatalf("Failed to create test KDL file: %v", err)
			}

			// Debug: check file content
			content, _ := os.ReadFile(kdlFile)
			t.Logf("File content: %q", string(content))

			// Run the compiled binary
			output, err := runKDLc(kdlFile)
			if err != nil {
				t.Fatalf("Failed to run kdlc: %v", err)
			}

			// Compare with expected output
			if !jsonEqualString(tt.expectedJSON, output) {
				t.Errorf("Output mismatch:\nExpected: %s\nActual: %s", tt.expectedJSON, output)
			}
		})
	}
}

// Test duplicate node handling (E2E version)
func TestDuplicateNodesE2E(t *testing.T) {
	// Check if binary exists before running E2E tests
	if err := checkBinaryExists(); err != nil {
		t.Skipf("Skipping E2E test: %v", err)
	}

	kdlContent := `item "sword" damage=10
item "shield" defense=5
item "sword" damage=15`

	// Create temporary KDL file in current directory to avoid include path issues
	kdlFile := "test_duplicate_e2e.kdl"
	defer os.Remove(kdlFile) // Clean up after test

	if err := os.WriteFile(kdlFile, []byte(kdlContent), 0644); err != nil {
		t.Fatalf("Failed to create test KDL file: %v", err)
	}

	// Run the compiled binary
	output, err := runKDLc(kdlFile)
	if err != nil {
		t.Fatalf("Failed to run kdlc: %v", err)
	}

	t.Logf("Binary output: %s", output)

	// Check if we have an array for duplicate nodes
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if item, ok := result["item"]; ok {
		if itemArray, ok := item.([]interface{}); ok {
			t.Logf("Found array with %d items", len(itemArray))
			if len(itemArray) != 3 {
				t.Errorf("Expected 3 items, got %d", len(itemArray))
			}
		} else {
			t.Errorf("Expected array for duplicate nodes, got: %T", item)
		}
	} else {
		t.Error("No 'item' key found in result")
	}
}

// Test include functionality using the compiled binary
func TestIncludeFunctionality(t *testing.T) {
	// Check if binary exists before running E2E tests
	if err := checkBinaryExists(); err != nil {
		t.Skipf("Skipping E2E test: %v", err)
	}

	tests := []struct {
		name         string
		files        map[string]string
		mainFile     string
		expectedJSON string
	}{
		{
			name: "basic include",
			files: map[string]string{
				"base.kdl": `config {
    version "1.0"
    theme "dark"
}`,
				"main.kdl": `@include "base.kdl"

scene "TestScene" {
    title "Main Scene"
}`,
			},
			mainFile: "main.kdl",
			expectedJSON: `{
  "config": {
    "theme": "dark",
    "version": "1.0"
  },
  "scene": {
    "arg1": "TestScene",
    "title": "Main Scene"
  }
}`,
		},
		{
			name: "nested includes",
			files: map[string]string{
				"config.kdl": `config {
    version "1.0"
}`,
				"ui.kdl": `ui {
    theme "dark"
}`,
				"main.kdl": `@include "config.kdl"
@include "ui.kdl"

scene "Main" {}`,
			},
			mainFile: "main.kdl",
			expectedJSON: `{
  "config": {
    "version": "1.0"
  },
  "scene": "Main",
  "ui": {
    "theme": "dark"
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and files
			tmpDir := t.TempDir()

			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", filename, err)
				}
			}

			// Run the compiled binary on main file
			mainFilePath := filepath.Join(tmpDir, tt.mainFile)
			output, err := runKDLc(mainFilePath)
			if err != nil {
				t.Fatalf("Failed to run kdlc: %v", err)
			}

			// Compare with expected output
			if !jsonEqualString(tt.expectedJSON, output) {
				t.Errorf("Output mismatch:\nExpected: %s\nActual: %s", tt.expectedJSON, output)
			}
		})
	}
}

// Test circular include detection using the compiled binary
func TestCircularIncludeDetection(t *testing.T) {
	// Check if binary exists before running E2E tests
	if err := checkBinaryExists(); err != nil {
		t.Skipf("Skipping E2E test: %v", err)
	}

	tests := []struct {
		name     string
		files    map[string]string
		mainFile string
		wantErr  bool
	}{
		{
			name: "simple circular reference",
			files: map[string]string{
				"file1.kdl": `@include "file2.kdl"
node1 "test1"`,
				"file2.kdl": `@include "file1.kdl"
node2 "test2"`,
			},
			mainFile: "file1.kdl",
			wantErr:  true,
		},
		{
			name: "self include",
			files: map[string]string{
				"self.kdl": `@include "self.kdl"
node "test"`,
			},
			mainFile: "self.kdl",
			wantErr:  true,
		},
		{
			name: "no circular reference",
			files: map[string]string{
				"base.kdl": `base "value"`,
				"main.kdl": `@include "base.kdl"
main "value"`,
			},
			mainFile: "main.kdl",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and files
			tmpDir := t.TempDir()

			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", filename, err)
				}
			}

			// Run the compiled binary
			mainFilePath := filepath.Join(tmpDir, tt.mainFile)
			_, err := runKDLc(mainFilePath)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error for circular include, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}

// Helper functions

// checkBinaryExists checks if the kdlc binary exists
func checkBinaryExists() error {
	if _, err := os.Stat("./kdlc"); os.IsNotExist(err) {
		return fmt.Errorf("kdlc binary not found. Please build it first with: go build -o kdlc main.go")
	}
	return nil
}

// jsonEqualString compares two JSON strings for equality
func jsonEqualString(expected, actual string) bool {
	var expectedJSON, actualJSON interface{}
	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		return false
	}
	return reflect.DeepEqual(expectedJSON, actualJSON)
}

// runKDLc runs the compiled kdlc binary with the given file
func runKDLc(filename string) (string, error) {
	return runKDLcWithArgs(filename, []string{})
}

// runKDLcWithArgs runs the compiled kdlc binary with the given file and arguments
func runKDLcWithArgs(filename string, args []string) (string, error) {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}

	// Build command with arguments
	cmdArgs := append(args, filename)
	cmd := exec.Command("./kdlc", cmdArgs...)
	cmd.Dir = wd // Set working directory explicitly
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout // Capture stderr too for error messages

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("kdlc execution failed: %v, output: %s", err, stdout.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Test parallel includes with duplicate nodes
func TestParallelIncludesWithDuplicates(t *testing.T) {
	// Check if binary exists before running E2E tests
	if err := checkBinaryExists(); err != nil {
		t.Skipf("Skipping E2E test: %v", err)
	}

	// Create test files in temporary directory
	tmpDir := t.TempDir()

	// File 1: Contains some items and buttons
	file1Content := `item "sword" damage=10
item "shield" defense=5
button "OK" x=100 y=100`

	file1Path := filepath.Join(tmpDir, "parallel_1.kdl")
	if err := os.WriteFile(file1Path, []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	// File 2: Contains duplicate items and buttons
	file2Content := `item "sword" damage=15
item "potion" healing=20
button "Cancel" x=200 y=100
button "OK" x=150 y=150`

	file2Path := filepath.Join(tmpDir, "parallel_2.kdl")
	if err := os.WriteFile(file2Path, []byte(file2Content), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Main file: Includes both files
	mainContent := `@include "parallel_1.kdl"
@include "parallel_2.kdl"

scene "ParallelIncludeScene" {
    title "Scene with Parallel Includes"
}`

	mainPath := filepath.Join(tmpDir, "parallel_main.kdl")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	// Run the compiled binary
	output, err := runKDLc(mainPath)
	if err != nil {
		t.Fatalf("Failed to run kdlc: %v", err)
	}

	t.Logf("Parallel include output: %s", output)

	// Parse the JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify item array (should have 4 items: 2 swords, 1 shield, 1 potion)
	if item, ok := result["item"]; ok {
		if itemArray, ok := item.([]interface{}); ok {
			if len(itemArray) != 4 {
				t.Errorf("Expected 4 items in item array, got %d", len(itemArray))
			}
			t.Logf("Found %d items in item array", len(itemArray))
		} else {
			t.Errorf("Expected array for item nodes, got: %T", item)
		}
	} else {
		t.Error("No 'item' key found in result")
	}

	// Verify button array (should have 3 items: 2 OK buttons, 1 Cancel button)
	if button, ok := result["button"]; ok {
		if buttonArray, ok := button.([]interface{}); ok {
			if len(buttonArray) != 3 {
				t.Errorf("Expected 3 buttons in button array, got %d", len(buttonArray))
			}
			t.Logf("Found %d buttons in button array", len(buttonArray))
		} else {
			t.Errorf("Expected array for button nodes, got: %T", button)
		}
	} else {
		t.Error("No 'button' key found in result")
	}

	// Verify scene object (should be single object)
	if scene, ok := result["scene"]; ok {
		if sceneObj, ok := scene.(map[string]interface{}); ok {
			t.Logf("Found scene object with keys: %v", getMapKeys(sceneObj))
		} else {
			t.Errorf("Expected object for scene node, got: %T", scene)
		}
	} else {
		t.Error("No 'scene' key found in result")
	}
}

// Test file-based conversion using simple.kdl
func TestFileBasedConversion(t *testing.T) {
	// Check if binary exists before running E2E tests
	if err := checkBinaryExists(); err != nil {
		t.Skipf("Skipping E2E test: %v", err)
	}

	tests := []struct {
		name         string
		kdlFile      string
		args         []string
		expectedJSON string
	}{
		{
			name:    "simple.kdl file",
			kdlFile: "testdata/simple.kdl",
			args:    []string{},
			expectedJSON: `{
  "scene": {
    "arg1": "SimpleScene",
    "node": {
      "arg1": "Button",
      "height": 50,
      "width": 200,
      "x": 100,
      "y": 100,
      "component": {
        "arg1": "Button",
        "text": "Click me"
      }
    }
  }
}`,
		},
		{
			name:    "simple.kdl with custom arg names",
			kdlFile: "testdata/simple.kdl",
			args:    []string{"-arg1=name"},
			expectedJSON: `{
  "scene": {
    "name": "SimpleScene",
    "node": {
      "component": {
        "name": "Button",
        "text": "Click me"
      },
      "height": 50,
      "name": "Button",
      "width": 200,
      "x": 100,
      "y": 100
    }
  }
}`,
		},
		{
			name:    "multi_args.kdl with custom arg names",
			kdlFile: "testdata/multi_args.kdl",
			args:    []string{"-arg1=first", "-arg2=second"},
			expectedJSON: `{
  "button": {
    "first": "OK",
    "second": "primary",
    "x": 100,
    "y": 100
  },
  "config": {
    "first": "app",
    "second": "settings",
    "theme": "dark",
    "version": "1.0"
  },
  "item": {
    "damage": 10,
    "first": "sword",
    "second": "weapon"
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the compiled binary on the test file with optional arguments
			output, err := runKDLcWithArgs(tt.kdlFile, tt.args)
			if err != nil {
				t.Fatalf("Failed to run kdlc on %s: %v", tt.kdlFile, err)
			}

			// Compare with expected output
			if !jsonEqualString(tt.expectedJSON, output) {
				t.Errorf("Output mismatch for %s:\nExpected: %s\nActual: %s", tt.kdlFile, tt.expectedJSON, output)
			}
		})
	}
}

// Test custom argument name mapping
func TestCustomArgumentNames(t *testing.T) {
	// Check if binary exists before running E2E tests
	if err := checkBinaryExists(); err != nil {
		t.Skipf("Skipping E2E test: %v", err)
	}

	tests := []struct {
		name         string
		args         []string
		kdlContent   string
		expectedJSON string
	}{
		{
			name: "custom arg1 name",
			args: []string{"-arg1=name"},
			kdlContent: `scene "TestScene" {
    node "Button" x=100 y=100
}`,
			expectedJSON: `{
  "scene": {
    "name": "TestScene",
    "node": {
      "name": "Button",
      "x": 100,
      "y": 100
    }
  }
}`,
		},
		{
			name: "multiple custom argument names",
			args: []string{"-arg1=action", "-arg2=target", "-arg3=x", "-arg4=y"},
			kdlContent: `command "move" "player" 100 200 {
    duration 0.5
}`,
			expectedJSON: `{
  "command": {
    "action": "move",
    "duration": 0.5,
    "target": "player",
    "x": 100,
    "y": 200
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary KDL file
			tmpDir := t.TempDir()
			kdlFile := filepath.Join(tmpDir, "test.kdl")
			if err := os.WriteFile(kdlFile, []byte(tt.kdlContent), 0644); err != nil {
				t.Fatalf("Failed to create test KDL file: %v", err)
			}

			// Run the compiled binary with custom arguments
			output, err := runKDLcWithArgs(kdlFile, tt.args)
			if err != nil {
				t.Fatalf("Failed to run kdlc with args %v: %v", tt.args, err)
			}

			// Compare with expected output
			if !jsonEqualString(tt.expectedJSON, output) {
				t.Errorf("Output mismatch for %s:\nExpected: %s\nActual: %s", tt.name, tt.expectedJSON, output)
			}
		})
	}
}

// Test duplicate node handling with direct parsing (no include processing)
func TestDuplicateNodesDirect(t *testing.T) {
	kdlContent := `item "sword" damage=10
item "shield" defense=5
item "sword" damage=15`

	// Parse KDL directly (simulate what happens in main.go)
	doc, err := kdl.Parse(strings.NewReader(kdlContent))
	if err != nil {
		t.Fatalf("Failed to parse KDL: %v", err)
	}

	t.Logf("Parsed %d nodes", len(doc.Nodes))
	for i, node := range doc.Nodes {
		t.Logf("Node %d: %s", i, node.Name.NodeNameString())
	}

	// Convert to JSON
	jsonData, err := convertKDLToJSON(doc)
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	t.Logf("JSON output: %s", string(jsonData))

	// Check if we have an array for duplicate nodes
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if item, ok := result["item"]; ok {
		if itemArray, ok := item.([]interface{}); ok {
			t.Logf("Found array with %d items", len(itemArray))
			if len(itemArray) != 3 {
				t.Errorf("Expected 3 items, got %d", len(itemArray))
			}
		} else {
			t.Errorf("Expected array for duplicate nodes, got: %T", item)
		}
	} else {
		t.Error("No 'item' key found in result")
	}
}

// Helper function to get map keys for logging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
