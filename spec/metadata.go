package spec

import (
    "iter"
    "maps"
)

type MetadataFilter interface {
    Include(key string) bool
}

type MetadataFilterFactory interface {
    BuildMetadataFilter(patterns []string, invert bool) (MetadataFilter, error)
}

type Metadata interface {
    Keys() iter.Seq[string]
    Get(key string) any
}

func NewMapMetadata(data map[string]any) Metadata {
    return &mapMetadata{
        data: data,
    }
}

type mapMetadata struct {
    data map[string]any
}

func (m *mapMetadata) Keys() iter.Seq[string] {
    return maps.Keys(m.data)
}

func (m *mapMetadata) Get(key string) any {
    return m.data[key]
}
