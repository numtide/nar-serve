package nar

import "strings"

// IsValidNodeName checks the name of a node
// it may not contain null bytes or slashes.
func IsValidNodeName(nodeName string) bool {
	return !strings.Contains(nodeName, "/") && !strings.ContainsAny(nodeName, "\u0000")
}

// PathIsLexicographicallyOrdered checks if two paths are lexicographically ordered component by component.
func PathIsLexicographicallyOrdered(path1 string, path2 string) bool {
	// n is the lower number of characters of the two paths.
	var n int
	if len(path1) < len(path2) {
		n = len(path1)
	} else {
		n = len(path2)
	}

	for i := 0; i < n; i++ {
		if path1[i] == path2[i] {
			continue
		}

		// A `/` terminates a path component, and entries are ordered by
		// component. So whichever path ends its component here sorts first,
		// whatever the other one continues with: `/a/b` precedes `/a-b`
		// because the entry `a` precedes the entry `a-b`. This has to be
		// decided before comparing the bytes themselves, since `/` is neither
		// the lowest nor the highest byte that can appear in a name.
		if path1[i] == '/' {
			return true
		}

		if path2[i] == '/' {
			return false
		}

		return path1[i] < path2[i]
	}

	// Cover cases like where path1 is a prefix of path2 (path1=/arp-foo path2=/arp)
	return len(path2) >= len(path1)
}
