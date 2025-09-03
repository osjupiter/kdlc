package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sblinch/kdl-go"
	"github.com/sblinch/kdl-go/document"
)

// Global configuration for argument name mapping
var argNameMap = map[int]string{
	1: "arg1",
	2: "arg2",
	3: "arg3",
	4: "arg4",
	5: "arg5",
}

// getArgName returns the configured name for the given argument index
func getArgName(index int) string {
	if name, exists := argNameMap[index]; exists {
		return name
	}
	return fmt.Sprintf("arg%d", index)
}

func main() {
	// Define command line flags
	arg1Name := flag.String("arg1", "arg1", "Name for the first argument")
	arg2Name := flag.String("arg2", "arg2", "Name for the second argument")
	arg3Name := flag.String("arg3", "arg3", "Name for the third argument")
	arg4Name := flag.String("arg4", "arg4", "Name for the fourth argument")
	arg5Name := flag.String("arg5", "arg5", "Name for the fifth argument")

	flag.Parse()

	// Update the argument name mapping
	argNameMap[1] = *arg1Name
	argNameMap[2] = *arg2Name
	argNameMap[3] = *arg3Name
	argNameMap[4] = *arg4Name
	argNameMap[5] = *arg5Name

	// Check if filename is provided
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <kdl-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	filename := flag.Arg(0)

	// Process includes and read KDL file
	data, err := processIncludes(filename, make(map[string]bool))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing includes: %v\n", err)
		os.Exit(1)
	}

	// Parse KDL
	doc, err := kdl.Parse(strings.NewReader(string(data)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing KDL: %v\n", err)
		os.Exit(1)
	}

	// Convert to JSON
	jsonData, err := convertKDLToJSON(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to JSON: %v\n", err)
		os.Exit(1)
	}

	// Output JSON
	fmt.Println(string(jsonData))
}

// processIncludes processes @include directives in KDL files
func processIncludes(filename string, included map[string]bool) (string, error) {
	// Check for circular includes
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %v", filename, err)
	}

	if included[absPath] {
		return "", fmt.Errorf("circular include detected: %s", filename)
	}
	included[absPath] = true

	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	content := string(data)

	// Check if file contains @include directives
	if !strings.Contains(content, "@include") {
		// No includes, return content as-is
		return content, nil
	}

	lines := strings.Split(content, "\n")
	var result []string

	// Process each line for @include directives
	includeRegex := regexp.MustCompile(`^\s*@include\s+"([^"]+)"`)

	for _, line := range lines {
		if matches := includeRegex.FindStringSubmatch(line); matches != nil {
			includeFile := matches[1]

			// Resolve relative path
			dir := filepath.Dir(filename)
			includePath := filepath.Join(dir, includeFile)

			// Process the included file
			includedContent, err := processIncludes(includePath, included)
			if err != nil {
				return "", fmt.Errorf("failed to process include %s: %v", includeFile, err)
			}

			// Add the included content
			result = append(result, includedContent)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n"), nil
}

func convertKDLToJSON(doc *document.Document) ([]byte, error) {
	// Convert KDL document to a map structure
	result := make(map[string]interface{})

	// Group nodes by name to handle duplicates
	nodeGroups := make(map[string][]*document.Node)
	for _, node := range doc.Nodes {
		key := node.Name.NodeNameString()
		nodeGroups[key] = append(nodeGroups[key], node)
	}

	// Process each group
	for key, nodes := range nodeGroups {
		if len(nodes) == 1 {
			// Single node
			result[key] = convertNodeToValue(nodes[0])
		} else {
			// Multiple nodes with same name - create array
			nodeArray := make([]interface{}, len(nodes))
			for i, node := range nodes {
				nodeArray[i] = convertNodeToValue(node)
			}
			result[key] = nodeArray
		}
	}

	return json.MarshalIndent(result, "", "  ")
}

func convertNodeToValue(node *document.Node) interface{} {
	// If node has children, convert to object
	if len(node.Children) > 0 {
		obj := make(map[string]interface{})

		// Add node arguments as configured argument names
		if len(node.Arguments) > 0 {
			for i, arg := range node.Arguments {
				argKey := getArgName(i + 1)
				obj[argKey] = convertValue(arg)
			}
		}

		// Add node properties directly (flatten the structure)
		if len(node.Properties) > 0 {
			for name, value := range node.Properties {
				obj[name] = convertValue(value)
			}
		}

		// Convert children
		childGroups := make(map[string][]*document.Node)
		for _, child := range node.Children {
			childKey := child.Name.NodeNameString()
			childGroups[childKey] = append(childGroups[childKey], child)
		}

		// Process child groups
		for childKey, childNodes := range childGroups {
			if len(childNodes) == 1 {
				obj[childKey] = convertNodeToValue(childNodes[0])
			} else {
				childArray := make([]interface{}, len(childNodes))
				for i, childNode := range childNodes {
					childArray[i] = convertNodeToValue(childNode)
				}
				obj[childKey] = childArray
			}
		}

		return obj
	}

	// If node has properties, convert to object with properties and arguments
	if len(node.Properties) > 0 {
		obj := make(map[string]interface{})

		// Add arguments as configured argument names if present
		if len(node.Arguments) > 0 {
			for i, arg := range node.Arguments {
				argKey := getArgName(i + 1)
				obj[argKey] = convertValue(arg)
			}
		}

		// Add properties directly (flatten the structure)
		for name, value := range node.Properties {
			obj[name] = convertValue(value)
		}

		return obj
	}

	// If node has multiple arguments, return as array
	if len(node.Arguments) > 1 {
		args := make([]interface{}, len(node.Arguments))
		for i, arg := range node.Arguments {
			args[i] = convertValue(arg)
		}
		return args
	}

	// If node has single argument, return the value directly
	if len(node.Arguments) == 1 {
		return convertValue(node.Arguments[0])
	}

	// Empty node
	return nil
}

func convertValue(value *document.Value) interface{} {
	if value == nil {
		return nil
	}

	resolved := value.ResolvedValue()
	switch v := resolved.(type) {
	case string:
		return v
	case int64:
		return v
	case float64:
		return v
	case bool:
		return v
	case nil:
		return nil
	default:
		return value.String()
	}
}
