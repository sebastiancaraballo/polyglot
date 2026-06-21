// Package polyglot embeds the course content bundled into the application binary.
//
// The embed directive lives at the module root so that the content/ tree can
// stay at the top level (Go embed cannot reference parent directories from
// within a subpackage).
package polyglot

import "embed"

// ContentFS holds the bundled course content (lessons, kana, guides) for every
// language pair under content/.
//
//go:embed all:content
var ContentFS embed.FS
