// Package pureterm holds the version of PureTerm. The release version is kept
// in the VERSION file, which the packages, the installer and the workflows read
// as well.
package pureterm

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version is the release version, for example "2.1.0".
var Version = strings.TrimSpace(versionFile)
