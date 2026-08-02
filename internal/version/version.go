package version

import "runtime/debug"

// Value is replaced by release builds through -ldflags when desired.
var Value = "devel"

// Current returns an explicit linker-provided version, the Go module build
// version for binaries installed from a tagged module, or "devel".
func Current() string {
	if Value != "" && Value != "devel" {
		return Value
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "devel"
}
