// skip : type definition https://github.com/open2b/scriggo/issues/194

// compile

// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type T struct {
	b bool
	string
}

func f() {
	var b bool
	var t T
	for {
		switch &t.b {
		case &b:
			if b {
			}
		}
	}
}

func main() { }