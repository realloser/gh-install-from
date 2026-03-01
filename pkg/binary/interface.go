package binary

// InstalledBinary represents an installed binary and its metadata
type InstalledBinary struct {
	Name           string
	Path           string
	Repository     string
	Version        string
	Host           string
	OriginalBinary string
}

// Manager defines the interface for binary management operations
type Manager interface {
	Install(repo string) error
	Update(repo string) error
	UpdateAll() error
	Remove(nameOrRepo string) error
	ListInstalled() ([]InstalledBinary, error)
	GetBinDir() string
	CheckUpdates(binaries []InstalledBinary) ([]UpdateCandidate, error)
}

// UpdateCandidate represents a binary that has an update available
type UpdateCandidate struct {
	InstalledBinary InstalledBinary
	LatestVersion   string
}
