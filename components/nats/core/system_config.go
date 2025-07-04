// Code generated by github.com/atombender/go-jsonschema, DO NOT EDIT.

package core

import (
	"encoding/json"
	"fmt"
)

type SystemConfig struct {
	// Optional authentication information for the NATS server.  If not provided, the
	// connection will be made without authentication.
	//
	Auth *SystemConfigAuth `json:"auth,omitempty" yaml:"auth,omitempty" mapstructure:"auth,omitempty"`

	// An optional name for the connection to distinguish it from others.
	Name string `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name,omitempty"`

	// Url of the NATS server to connect to.  Multiple URLs can be specified by
	// separating them with commas. If an item of the list contains commas it will  be
	// expanded into multiple URLs.
	// Examples:
	//   - nats://demo.nats.io:4222
	//   - nats://server-1:4222,nats://server-2:4222
	//
	Url string `json:"url" yaml:"url" mapstructure:"url"`
}

// Optional authentication information for the NATS server.  If not provided, the
// connection will be made without authentication.
type SystemConfigAuth struct {
	// The user JWT token. This is a sensitive field and you may want to use
	// environment variables instead of defining a constant value.
	//
	Jwt string `json:"jwt" yaml:"jwt" mapstructure:"jwt"`

	// The user seed.  This is a sensitive field and you may want to use environment
	// variables instead of defining a constant value.
	//
	Seed string `json:"seed" yaml:"seed" mapstructure:"seed"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SystemConfigAuth) UnmarshalJSON(value []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(value, &raw); err != nil {
		return err
	}
	if _, ok := raw["jwt"]; raw != nil && !ok {
		return fmt.Errorf("field jwt in SystemConfigAuth: required")
	}
	if _, ok := raw["seed"]; raw != nil && !ok {
		return fmt.Errorf("field seed in SystemConfigAuth: required")
	}
	type Plain SystemConfigAuth
	var plain Plain
	if err := json.Unmarshal(value, &plain); err != nil {
		return err
	}
	*j = SystemConfigAuth(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SystemConfig) UnmarshalJSON(value []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(value, &raw); err != nil {
		return err
	}
	type Plain SystemConfig
	var plain Plain
	if err := json.Unmarshal(value, &plain); err != nil {
		return err
	}
	if v, ok := raw["name"]; !ok || v == nil {
		plain.Name = "wombat"
	}
	if v, ok := raw["url"]; !ok || v == nil {
		plain.Url = "nats://localhost:4222"
	}
	*j = SystemConfig(plain)
	return nil
}
