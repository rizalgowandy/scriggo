// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"scriggo"
	"scriggo/runtime"
)

const usage = "usage: %s [-S] [-mem 250K] [-time 50ms] filename\n"

var packages scriggo.Packages
var Main *scriggo.Package

func renderPanics(p *runtime.Panic) string {
	var msg string
	for ; p != nil; p = p.Next() {
		msg = "\n" + msg
		if p.Recovered() {
			msg = " [recovered]" + msg
		}
		msg = p.String() + msg
		if p.Next() != nil {
			msg = "\tpanic: " + msg
		}
	}
	return msg
}

func run() {

	var asm = flag.Bool("S", false, "print assembly listing")
	var timeout = flag.String("time", "", "limit the execution time; zero is no limit")
	var mem = flag.String("mem", "", "limit the allocable memory; zero is no limit")

	flag.Parse()

	var loadOptions = &scriggo.LoadOptions{}
	var runOptions = &scriggo.RunOptions{}

	if *timeout != "" {
		d, err := time.ParseDuration(*timeout)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, usage, os.Args[0])
			flag.PrintDefaults()
			os.Exit(1)
		}
		if d != 0 {
			var cancel context.CancelFunc
			runOptions.Context, cancel = context.WithTimeout(context.Background(), d)
			defer cancel()
		}
	}

	if *mem != "" {
		var unit = (*mem)[len(*mem)-1]
		if unit > 'Z' {
			unit -= 'z' - 'Z'
		}
		switch unit {
		case 'B', 'K', 'M', 'G':
			*mem = (*mem)[:len(*mem)-1]
		}
		max, err := strconv.Atoi(*mem)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, usage, os.Args[0])
			flag.PrintDefaults()
			os.Exit(1)
		}
		switch unit {
		case 'K':
			max *= 1024
		case 'M':
			max *= 1024 * 1024
		case 'G':
			max *= 1024 * 1024 * 1024
		}
		loadOptions.LimitMemory = true
		runOptions.MemoryLimiter = scriggo.NewSingleMemoryLimiter(max)
	}

	var args = flag.Args()

	if len(args) != 1 {
		_, _ = fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	file := args[0]
	ext := filepath.Ext(file)
	if ext != ".go" {
		fmt.Printf("%s: extension must be \".go\"\n", file)
		os.Exit(1)
	}

	absFile, err := filepath.Abs(file)
	if err != nil {
		fmt.Printf("%s: %s\n", file, err)
		os.Exit(1)
	}

	main, err := ioutil.ReadFile(absFile)
	if err != nil {
		panic(err)
	}
	program, err := scriggo.Load(bytes.NewReader(main), scriggo.Loaders(packages), loadOptions)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
		os.Exit(2)
	}
	if *asm {
		_, err := program.Disassemble(os.Stdout, "main")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
			os.Exit(2)
		}
	} else {
		code, err := program.Run(runOptions)
		if err != nil {
			if p, ok := err.(*runtime.Panic); ok {
				panic(renderPanics(p))
			}
			if err == context.DeadlineExceeded {
				err = errors.New("process took too long")
			}
			_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
			os.Exit(2)
		}
		os.Exit(code)
	}
	os.Exit(0)
}
