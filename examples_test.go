// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"fmt"
	"log"
	"os"

	"github.com/open2b/scriggo/native"
)

func ExampleBuild() {
	fsys := Files{
		"main.go": []byte(`
			package main

			func main() { }
		`),
	}
	_, err := Build(fsys, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleProgram_Run() {
	fsys := Files{
		"main.go": []byte(`
			package main

			import "fmt"

			func main() {
				fmt.Println("Hello, I'm Scriggo!")
			}
		`),
	}
	opts := &BuildOptions{
		Packages: native.Packages{
			"fmt": native.Package{
				Name: "fmt",
				Declarations: native.Declarations{
					"Println": fmt.Println,
				},
			},
		},
	}
	program, err := Build(fsys, opts)
	if err != nil {
		log.Fatal(err)
	}
	err = program.Run(nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Hello, I'm Scriggo!
}

func ExampleBuildTemplate() {
	fsys := Files{
		"index.html": []byte(`{% name := "Scriggo" %}Hello, {{ name }}!`),
	}
	_, err := BuildTemplate(fsys, "index.html", nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleTemplate_Run() {
	fsys := Files{
		"index.html": []byte(`{% name := "Scriggo" %}Hello, {{ name }}!`),
	}
	template, err := BuildTemplate(fsys, "index.html", nil)
	if err != nil {
		log.Fatal(err)
	}
	err = template.Run(os.Stdout, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Hello, Scriggo!
}

func ExampleHTMLEscape() {
	fmt.Println(HTMLEscape("Rock & Roll!"))
	// Output:
	// Rock &amp; Roll!
}

func ExampleBuildError() {
	fsys := Files{
		"index.html": []byte(`{{ 42 + "hello" }}`),
	}
	_, err := BuildTemplate(fsys, "index.html", nil)
	if err != nil {
		fmt.Printf("Error has type %T\n", err)
		fmt.Printf("Error message is: %s\n", err.(*BuildError).Message())
		fmt.Printf("Error path is: %s\n", err.(*BuildError).Path())
	}
	// Output:
	// Error has type *scriggo.BuildError
	// Error message is: invalid operation: 42 + "hello" (mismatched types int and string)
	// Error path is: index.html
}
