// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helm

import "strings"

// splitKeyPath splits a Helm-style value path into key segments,
// respecting backslash-escaped dots (\.) as literal dots within a key.
// This matches the escaping convention used by Helm's strvals package.
//
// Examples:
//
//	splitKeyPath("server.config")          => ["server", "config"]
//	splitKeyPath("server.oidc\\.config")   => ["server", "oidc.config"]
//	splitKeyPath("a\\.b\\.c")              => ["a.b.c"]
//	splitKeyPath("simple")                 => ["simple"]
func splitKeyPath(path string) []string {
	var segments []string
	var current strings.Builder

	i := 0
	for i < len(path) {
		if path[i] == '\\' && i+1 < len(path) && path[i+1] == '.' {
			// Escaped dot: write literal dot, skip both characters
			current.WriteByte('.')
			i += 2
		} else if path[i] == '.' {
			// Unescaped dot: segment boundary
			segments = append(segments, current.String())
			current.Reset()
			i++
		} else {
			current.WriteByte(path[i])
			i++
		}
	}
	segments = append(segments, current.String())

	return segments
}
