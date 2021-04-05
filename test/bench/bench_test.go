// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
)

func BenchmarkRun(b *testing.B) {
	programs, err := build()
	if err != nil {
		b.Fatal(err)
	}
	for _, program := range programs {
		b.Run(program.name, func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				_, err := program.code.Run(nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
