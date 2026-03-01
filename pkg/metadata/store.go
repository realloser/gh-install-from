// Package metadata defines the MetadataStore interface for binary metadata
package metadata

// MetadataStore defines the interface for binary metadata storage
type MetadataStore interface {
	// Store saves metadata for an installed binary
	Store(meta *BinaryMetadata) error
	// Load retrieves metadata for a binary by name
	Load(binaryName string) (*BinaryMetadata, error)
	// Delete removes metadata for a binary
	Delete(binaryName string) error
	// List returns all stored metadata
	List() ([]*BinaryMetadata, error)
}
