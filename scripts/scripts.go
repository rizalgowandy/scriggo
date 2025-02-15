// Copyright 2020 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scripts allows the Go code to be executed as a script.
//
// Package scripts is EXPERIMENTAL. There is no guarantee that it will be
// included in the 1.0 version of Scriggo and could be removed at any time.
package scripts

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/internal/compiler"
	"github.com/open2b/scriggo/internal/runtime"
	"github.com/open2b/scriggo/native"
)

// BuildOptions contains options for building scripts.
type BuildOptions struct {

	// AllowGoStmt, when true, allows the use of the go statement.
	AllowGoStmt bool

	// Packages is a package importer that makes native packages available
	// in scripts through the import statement.
	Packages native.Importer

	// Globals declares constants, types, variables, functions and packages
	// that are accessible from the code in the script.
	Globals native.Declarations
}

// RunOptions are the run options.
type RunOptions struct {

	// Context is a context that can be read by native functions and methods
	// via the Context method of native.Env. Canceling the context, the
	// execution is terminated and the Run method returns Context.Err().
	Context context.Context

	// Print is called by the print and println builtins to print values.
	// If it is nil, the print and println builtins format their arguments as
	// expected and write the result to standard error.
	Print scriggo.PrintFunc
}

// Script is a script compiled with the Build function.
type Script struct {
	fn      *runtime.Function
	typeof  runtime.TypeOfFunc
	globals []compiler.Global
}

// Build builds a script reading the source code from src.
//
// If a build error occurs, it returns a *BuildError.
func Build(src io.Reader, options *BuildOptions) (*Script, error) {
	co := compiler.Options{}
	if options != nil {
		co.Globals = options.Globals
		co.AllowGoStmt = options.AllowGoStmt
		co.Importer = options.Packages
	}
	code, err := compiler.BuildScript(src, co)
	if err != nil {
		if e, ok := err.(compiler.Error); ok {
			err = &BuildError{err: e}
		}
		return nil, err
	}
	return &Script{fn: code.Main, globals: code.Globals, typeof: code.TypeOf}, nil
}

// Disassemble disassembles the script and returns its assembly code.
func (p *Script) Disassemble() []byte {
	assemblies := compiler.Disassemble(p.fn, p.globals, 0)
	return assemblies["main"]
}

// Run starts the script and waits for it to complete. vars contains the
// values of the global variables.
//
// If the executed script panics, and it is not recovered, Run returns a
// *PanicError.
//
// If the Stop method of native.Env is called, Run returns the argument passed
// to Stop.
//
// If the exit builtin is called, Run returns a scriggo.ExitError with the
// exit code.
//
// If the Fatal method of native.Env is called, Run panics with the argument
// passed to Fatal.
//
// If the context has been canceled, Run returns the error returned by the Err
// method of the context.
func (p *Script) Run(vars map[string]interface{}, options *RunOptions) error {
	vm := runtime.NewVM()
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.Print != nil {
			vm.SetPrint(runtime.PrintFunc(options.Print))
		}
	}
	err := vm.Run(p.fn, p.typeof, initGlobalVariables(p.globals, vars))
	if err != nil {
		if p, ok := err.(*runtime.PanicError); ok {
			err = &PanicError{p}
		}
		return err
	}
	return nil
}

var emptyInit = map[string]interface{}{}

// initGlobalVariables initializes the global variables and returns their
// values. It panics if init is not valid.
//
// This function is a copy of the function in the scriggo package.
func initGlobalVariables(variables []compiler.Global, init map[string]interface{}) []reflect.Value {
	n := len(variables)
	if n == 0 {
		return nil
	}
	if init == nil {
		init = emptyInit
	}
	values := make([]reflect.Value, n)
	for i, variable := range variables {
		if variable.Pkg == "main" {
			if value, ok := init[variable.Name]; ok {
				if variable.Value.IsValid() {
					panic(fmt.Sprintf("variable %q already initialized", variable.Name))
				}
				if value == nil {
					panic(fmt.Sprintf("variable initializer %q cannot be nil", variable.Name))
				}
				val := reflect.ValueOf(value)
				if typ := val.Type(); typ == variable.Type {
					v := reflect.New(typ).Elem()
					v.Set(val)
					values[i] = v
				} else {
					if typ.Kind() != reflect.Ptr || typ.Elem() != variable.Type {
						panic(fmt.Sprintf("variable initializer %q must have type %s or %s, but have %s",
							variable.Name, variable.Type, reflect.PtrTo(variable.Type), typ))
					}
					if val.IsNil() {
						panic(fmt.Sprintf("variable initializer %q cannot be a nil pointer", variable.Name))
					}
					values[i] = reflect.ValueOf(value).Elem()
				}
				continue
			}
		}
		if variable.Value.IsValid() {
			values[i] = variable.Value
		} else {
			values[i] = reflect.New(variable.Type).Elem()
		}
	}
	return values
}
