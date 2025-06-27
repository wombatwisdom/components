package spec_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/framework/spec"
)

var _ = Describe("ConfigSchema", func() {
	var schema spec.ConfigSchema

	BeforeEach(func() {
		schema = spec.NewConfigSchema()
	})

	It("should create empty schema", func() {
		jsonStr, err := schema.ToJSON()
		Expect(err).NotTo(HaveOccurred())

		var result map[string]any
		err = json.Unmarshal([]byte(jsonStr), &result)
		Expect(err).NotTo(HaveOccurred())

		Expect(result["type"]).To(Equal("object"))
		Expect(result["properties"]).To(Equal(map[string]any{}))
		Expect(result["required"]).To(Equal([]any{}))
	})

	It("should add string field", func() {
		schema.AddField(spec.SchemaField{
			Name:        "url",
			Type:        "string",
			Description: "Connection URL",
			Required:    true,
			Default:     "http://localhost",
			Examples:    []any{"http://example.com", "https://api.example.com"},
		})

		jsonStr, err := schema.ToJSON()
		Expect(err).NotTo(HaveOccurred())

		var result map[string]any
		err = json.Unmarshal([]byte(jsonStr), &result)
		Expect(err).NotTo(HaveOccurred())

		properties := result["properties"].(map[string]any)
		urlField := properties["url"].(map[string]any)

		Expect(urlField["type"]).To(Equal("string"))
		Expect(urlField["description"]).To(Equal("Connection URL"))
		Expect(urlField["default"]).To(Equal("http://localhost"))
		Expect(urlField["examples"]).To(ConsistOf("http://example.com", "https://api.example.com"))

		required := result["required"].([]any)
		Expect(required).To(ContainElement("url"))
	})

	It("should add multiple fields", func() {
		schema.AddField(spec.SchemaField{
			Name:        "host",
			Type:        "string",
			Description: "Server host",
			Required:    true,
		}).AddField(spec.SchemaField{
			Name:        "port",
			Type:        "integer",
			Description: "Server port",
			Required:    false,
			Default:     8080,
		}).AddField(spec.SchemaField{
			Name:        "timeout",
			Type:        "number",
			Description: "Connection timeout in seconds",
			Required:    false,
			Default:     30.0,
		})

		jsonStr, err := schema.ToJSON()
		Expect(err).NotTo(HaveOccurred())

		var result map[string]any
		err = json.Unmarshal([]byte(jsonStr), &result)
		Expect(err).NotTo(HaveOccurred())

		properties := result["properties"].(map[string]any)
		Expect(properties).To(HaveLen(3))

		// Check host field (required)
		hostField := properties["host"].(map[string]any)
		Expect(hostField["type"]).To(Equal("string"))
		Expect(hostField["description"]).To(Equal("Server host"))
		Expect(hostField).NotTo(HaveKey("default"))

		// Check port field (optional with default)
		portField := properties["port"].(map[string]any)
		Expect(portField["type"]).To(Equal("integer"))
		Expect(portField["default"]).To(Equal(float64(8080))) // JSON numbers are float64

		// Check timeout field (optional with default)
		timeoutField := properties["timeout"].(map[string]any)
		Expect(timeoutField["type"]).To(Equal("number"))
		Expect(timeoutField["default"]).To(Equal(30.0))

		// Check required fields
		required := result["required"].([]any)
		Expect(required).To(ContainElement("host"))
		Expect(required).NotTo(ContainElement("port"))
		Expect(required).NotTo(ContainElement("timeout"))
	})

	It("should handle field with no examples", func() {
		schema.AddField(spec.SchemaField{
			Name:        "simple",
			Type:        "string",
			Description: "Simple field",
			Required:    false,
		})

		jsonStr, err := schema.ToJSON()
		Expect(err).NotTo(HaveOccurred())

		var result map[string]any
		err = json.Unmarshal([]byte(jsonStr), &result)
		Expect(err).NotTo(HaveOccurred())

		properties := result["properties"].(map[string]any)
		simpleField := properties["simple"].(map[string]any)

		Expect(simpleField).NotTo(HaveKey("examples"))
		Expect(simpleField).NotTo(HaveKey("default"))
	})

	It("should overwrite field if added twice", func() {
		schema.AddField(spec.SchemaField{
			Name:        "field",
			Type:        "string",
			Description: "First description",
			Required:    true,
		}).AddField(spec.SchemaField{
			Name:        "field",
			Type:        "integer",
			Description: "Second description",
			Required:    false,
			Default:     42,
		})

		jsonStr, err := schema.ToJSON()
		Expect(err).NotTo(HaveOccurred())

		var result map[string]any
		err = json.Unmarshal([]byte(jsonStr), &result)
		Expect(err).NotTo(HaveOccurred())

		properties := result["properties"].(map[string]any)
		field := properties["field"].(map[string]any)

		// Should have the second field's values
		Expect(field["type"]).To(Equal("integer"))
		Expect(field["description"]).To(Equal("Second description"))
		Expect(field["default"]).To(Equal(float64(42)))

		// Should not be required (second field was not required)
		required := result["required"].([]any)
		Expect(required).NotTo(ContainElement("field"))
	})
})
