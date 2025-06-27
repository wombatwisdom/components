package spec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/framework/spec"
)

var _ = Describe("Config", func() {
	Describe("YamlConfig", func() {
		It("should decode simple YAML", func() {
			yamlContent := `
name: test
port: 8080
enabled: true
`
			cfg := spec.NewYamlConfig(yamlContent)

			var result map[string]any
			err := cfg.Decode(&result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result["name"]).To(Equal("test"))
			Expect(result["port"]).To(Equal(8080))
			Expect(result["enabled"]).To(Equal(true))
		})

		It("should decode into struct", func() {
			yamlContent := `
host: localhost
port: 9090
timeout: 30
`
			cfg := spec.NewYamlConfig(yamlContent)

			type TestConfig struct {
				Host    string `yaml:"host"`
				Port    int    `yaml:"port"`
				Timeout int    `yaml:"timeout"`
			}

			var result TestConfig
			err := cfg.Decode(&result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Host).To(Equal("localhost"))
			Expect(result.Port).To(Equal(9090))
			Expect(result.Timeout).To(Equal(30))
		})

		It("should handle string replacements", func() {
			yamlContent := `
host: __HOST__
port: __PORT__
`
			cfg := spec.NewYamlConfig(yamlContent, "__HOST__", "example.com", "__PORT__", "3000")

			var result map[string]any
			err := cfg.Decode(&result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result["host"]).To(Equal("example.com"))
			Expect(result["port"]).To(Equal(3000)) // YAML will parse 3000 as int
		})

		It("should trim whitespace", func() {
			yamlContent := `

name: test
value: 42

`
			cfg := spec.NewYamlConfig(yamlContent)

			var result map[string]any
			err := cfg.Decode(&result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result["name"]).To(Equal("test"))
			Expect(result["value"]).To(Equal(42))
		})

		It("should return error for invalid YAML", func() {
			yamlContent := `
name: test
invalid: [unclosed
`
			cfg := spec.NewYamlConfig(yamlContent)

			var result map[string]any
			err := cfg.Decode(&result)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("MapConfig", func() {
		It("should decode simple map", func() {
			data := map[string]any{
				"host":    "localhost",
				"port":    8080,
				"enabled": true,
			}
			cfg := spec.NewMapConfig(data)

			var result map[string]any
			err := cfg.Decode(&result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result["host"]).To(Equal("localhost"))
			Expect(result["port"]).To(Equal(8080))
			Expect(result["enabled"]).To(Equal(true))
		})

		It("should decode into struct", func() {
			data := map[string]any{
				"host":    "example.com",
				"port":    9090,
				"timeout": 30,
			}
			cfg := spec.NewMapConfig(data)

			type TestConfig struct {
				Host    string `mapstructure:"host"`
				Port    int    `mapstructure:"port"`
				Timeout int    `mapstructure:"timeout"`
			}

			var result TestConfig
			err := cfg.Decode(&result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Host).To(Equal("example.com"))
			Expect(result.Port).To(Equal(9090))
			Expect(result.Timeout).To(Equal(30))
		})

		It("should handle nested structures", func() {
			data := map[string]any{
				"database": map[string]any{
					"host": "db.example.com",
					"port": 5432,
				},
				"cache": map[string]any{
					"host": "cache.example.com",
					"port": 6379,
				},
			}
			cfg := spec.NewMapConfig(data)

			type DatabaseConfig struct {
				Host string `mapstructure:"host"`
				Port int    `mapstructure:"port"`
			}

			type TestConfig struct {
				Database DatabaseConfig `mapstructure:"database"`
				Cache    DatabaseConfig `mapstructure:"cache"`
			}

			var result TestConfig
			err := cfg.Decode(&result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Database.Host).To(Equal("db.example.com"))
			Expect(result.Database.Port).To(Equal(5432))
			Expect(result.Cache.Host).To(Equal("cache.example.com"))
			Expect(result.Cache.Port).To(Equal(6379))
		})

		It("should handle empty map", func() {
			data := map[string]any{}
			cfg := spec.NewMapConfig(data)

			var result map[string]any
			err := cfg.Decode(&result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(BeEmpty())
		})

		It("should handle type conversion errors gracefully", func() {
			data := map[string]any{
				"port": "not-a-number", // String where int expected
			}
			cfg := spec.NewMapConfig(data)

			type TestConfig struct {
				Port int `mapstructure:"port"`
			}

			var result TestConfig
			err := cfg.Decode(&result)
			// mapstructure does error on type conversion failures
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected type 'int'"))
		})
	})
})
