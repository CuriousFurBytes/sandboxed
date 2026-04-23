package assets

import _ "embed"

// DefaultContainerfile is the bundled Fedora-based sandbox image definition.
// It is extracted to $XDG_DATA_HOME/sandbox/Containerfile on first `sbx build`.
//
//go:embed Containerfile
var DefaultContainerfile []byte
