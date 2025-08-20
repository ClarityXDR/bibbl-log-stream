package version

// Variables populated via -ldflags at build time.
// Example:
//   go build -ldflags "-X 'bibbl/internal/version.Version=1.0.0' -X 'bibbl/internal/version.Commit=$(git rev-parse --short HEAD)' -X 'bibbl/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
var (
    Version = "dev"
    Commit  = ""
    Date    = ""
)

// Full returns a human friendly version string.
func Full() string {
    if Commit == "" {
        return Version
    }
    return Version + "+" + Commit
}
