package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ComponentConfig struct {
	Package            string
	Module             string
	SystemName         string
	ServiceName        string
	Description        string
	ClientType         string
	ConfigExample      string
	ValidConfigExample string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run scripts/generate-component.go <component-name>")
		fmt.Println("Example: go run scripts/generate-component.go redis")
		os.Exit(1)
	}

	componentName := strings.ToLower(os.Args[1])

	// Interactive configuration
	config := ComponentConfig{
		Package:    componentName,
		Module:     "github.com/wombatwisdom/components",
		SystemName: cases.Title(language.English).String(componentName),
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Service name (default: %s): ", cases.Title(language.English).String(componentName))
	if serviceName, _ := reader.ReadString('\n'); strings.TrimSpace(serviceName) != "" {
		config.ServiceName = strings.TrimSpace(serviceName)
	} else {
		config.ServiceName = cases.Title(language.English).String(componentName)
	}

	fmt.Print("Description: ")
	description, _ := reader.ReadString('\n')
	config.Description = strings.TrimSpace(description)

	fmt.Print("Client type (e.g., *redis.Client): ")
	clientType, _ := reader.ReadString('\n')
	config.ClientType = strings.TrimSpace(clientType)

	fmt.Println("Configuration example for invalid test:")
	fmt.Print("> ")
	configExample, _ := reader.ReadString('\n')
	config.ConfigExample = strings.TrimSpace(configExample)

	fmt.Println("Valid configuration example:")
	fmt.Print("> ")
	validConfigExample, _ := reader.ReadString('\n')
	config.ValidConfigExample = strings.TrimSpace(validConfigExample)

	// Create component directory at repository root
	componentDir := filepath.Join("../../", componentName)
	if err := os.MkdirAll(componentDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Template files to generate
	templates := map[string]string{
		"system.go":      "templates/component/system.go.tmpl",
		"input.go":       "templates/component/input.go.tmpl",
		"output.go":      "templates/component/output.go.tmpl",
		"system_test.go": "templates/component/system_test.go.tmpl",
		"Taskfile.yml":   "templates/component/Taskfile.yml.tmpl",
	}

	// Generate files
	for filename, templatePath := range templates {
		if err := generateFile(templatePath, filepath.Join(componentDir, filename), config); err != nil {
			fmt.Printf("Error generating %s: %v\n", filename, err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s/%s\n", componentName, filename)
	}

	// Generate schema files
	schemas := []string{
		"system_config.schema.yaml",
		"input_config.schema.yaml",
		"output_config.schema.yaml",
	}

	for _, schema := range schemas {
		schemaPath := filepath.Join(componentDir, schema)
		if err := generateSchemaFile(schemaPath, config); err != nil {
			fmt.Printf("Error generating %s: %v\n", schema, err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s/%s\n", componentName, schema)
	}

	// Generate test suite
	suiteFile := filepath.Join(componentDir, fmt.Sprintf("%s_suite_test.go", componentName))
	if err := generateSuiteFile(suiteFile, config); err != nil {
		fmt.Printf("Error generating test suite: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated: %s/%s_suite_test.go\n", componentName, componentName)

	fmt.Printf("\n✅ Component '%s' generated successfully!\n", componentName)
	fmt.Printf("📁 Directory: %s/\n", componentName)
	fmt.Printf("🔧 Next steps:\n")
	fmt.Printf("   1. cd %s\n", componentName)
	fmt.Printf("   2. task models:generate\n")
	fmt.Printf("   3. Implement TODO items in the generated files\n")
	fmt.Printf("   4. task test\n")
}

func generateFile(templatePath, outputPath string, config ComponentConfig) error {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	return tmpl.Execute(file, config)
}

func generateSchemaFile(outputPath string, config ComponentConfig) error {
	schemaContent := fmt.Sprintf(`$schema: "https://json-schema.org/draft/2020-12/schema"
$id: "https://wombatwisdom.com/schemas/%s/%s"
title: "%s Configuration"
description: "Configuration for %s %s"
type: object
properties:
  # TODO: Define configuration properties
additionalProperties: false
`, config.Package, filepath.Base(outputPath), config.ServiceName, config.ServiceName, strings.TrimSuffix(filepath.Base(outputPath), ".schema.yaml"))

	return os.WriteFile(outputPath, []byte(schemaContent), 0644)
}

func generateSuiteFile(outputPath string, config ComponentConfig) error {
	suiteContent := fmt.Sprintf(`package %s_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Test%s(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "%s Suite")
}
`, config.Package, cases.Title(language.English).String(config.Package), cases.Title(language.English).String(config.ServiceName))

	return os.WriteFile(outputPath, []byte(suiteContent), 0644)
}
