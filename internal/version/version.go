package version

// Version is set at build time via -ldflags "-X github.com/abunjevac/devtabs/internal/version.Version=vX.Y.Z".
// Falls back to "dev" for local builds without a tag.
var Version = "dev"
