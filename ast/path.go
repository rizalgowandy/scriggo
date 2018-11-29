// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast

import (
	"strings"
	"unicode/utf8"
)

// IsPath indicates whether path is valid as an extend, include and show path.
// These are valid paths: "/a.a", "/a/a.a", "a.a", "a/a.a", "../a.a", "a/../a.a".
// These are invalid paths: "", "/", "a", "aa" "aa.", "a/", "..", "a/..".
func IsPath(path string) bool {
	// Must have at least one character and do not end with '/'.
	if path == "" || path[len(path)-1] == '/' {
		return false
	}
	// Splits the path in the various names.
	var names = strings.Split(path, "/")
	// First names must be directories or '..'.
	for i, name := range names[:len(names)-1] {
		// If the first name is empty, path starts with '/'.
		if i == 0 && name == "" {
			continue
		}
		if name != ".." && !isDirName(name) {
			return false
		}
	}
	// Last name must be a file.
	return isFileName(names[len(names)-1])
}

func isDirName(name string) bool {
	// Must be at least one character long and less than 256.
	if name == "" || utf8.RuneCountInString(name) >= 256 {
		return false
	}
	// Should not be '.' and must not contain '..'.
	if name == "." || strings.Contains(name, "..") {
		return false
	}
	// First and last character should not be spaces.
	if name[0] == ' ' || name[len(name)-1] == ' ' {
		return false
	}
	return !isWindowsReservedName(name)
}

func isFileName(name string) bool {
	// Must be at least 3 characters long and less than 256.
	var length = utf8.RuneCountInString(name)
	if length <= 2 || length >= 256 {
		return false
	}
	// First and the last character can not be a point.
	if name[0] == '.' || name[len(name)-1] == '.' {
		return false
	}
	// Extension must be present.
	name = strings.ToLower(name)
	var dot = strings.LastIndexByte(name, '.')
	var ext = name[dot+1:]
	if strings.IndexByte(ext, '.') >= 0 {
		return false
	}
	// First and last character should not be spaces.
	if name[0] == ' ' || name[len(name)-1] == ' ' {
		return false
	}
	return !isWindowsReservedName(name)
}

// isWindowsReservedName indicates if name is a reserved file name on Windows.
// See https://docs.microsoft.com/en-us/windows/desktop/fileio/naming-a-file
func isWindowsReservedName(name string) bool {
	const DEL = '\x7f'
	for i := 0; i < len(name); i++ {
		switch c := name[i]; c {
		case '"', '*', '/', ':', '<', '>', '?', '\\', '|', DEL:
			return true
		default:
			if c <= '\x1f' {
				return true
			}
		}
	}
	switch name {
	case "con", "prn", "aux", "nul",
		"com0", "com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8",
		"com9", "lpt0", "lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7",
		"lpt8", "lpt9":
		return true
	}
	if len(name) >= 4 {
		switch name[0:4] {
		case "con.", "prn.", "aux.", "nul.":
			return true
		}
		if len(name) >= 5 {
			switch name[0:5] {
			case "com0.", "com1.", "com2.", "com3.", "com4.", "com5.", "com6.",
				"com7.", "com8.", "com9.", "lpt0.", "lpt1.", "lpt2.", "lpt3.",
				"lpt4.", "lpt5.", "lpt6.", "lpt7.", "lpt8.", "lpt9.":
				return true
			}
		}
	}
	return false
}
