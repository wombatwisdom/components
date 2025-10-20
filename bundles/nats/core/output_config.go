package core

import (
	"github.com/wombatwisdom/components/framework/spec"
)

type OutputConfig struct {
	// Optional metadata filters
	//
	MetadataFilter spec.MetadataFilter

	// The subject to publish to. The subject may not contain wildcards, but may
	// contain variables that are extracted from the message being processed.
	//
	Subject spec.Expression
}
