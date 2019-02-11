// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cockroachdb/apd"

	"open2b/template/ast"
)

var decimalMaxInt = apd.New(maxInt, 0)
var decimalMinInt = apd.New(minInt, 0)
var decimalMaxRune = apd.New(2147483647, 0)
var decimalMinRune = apd.New(-2147483648, 0)
var decimalMaxByte = apd.New(255, 0)
var decimalMinByte = apd.New(0, 0)
var decimal1 = apd.New(1, 0)
var decimalMod8 = apd.New(256, 0)
var decimalMod32 = apd.New(4294967296, 0)

var decimalModInt *apd.Decimal

func init() {
	if 18446744073709551615 == uint64(^uint(0)) {
		decimalModInt, _, _ = apd.NewFromString("18446744073709551616")
	} else {
		decimalModInt, _, _ = apd.NewFromString("4294967296")
	}
}

// Error records a rendering error with the path and the position where
// the error occurred.
type Error struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (err *Error) Error() string {
	return fmt.Sprintf("%s:%s: %s", err.Path, err.Pos, err.Err)
}

// errBreak returned from rendering the "break" statement.
// It is managed by the innermost "for" statement.
var errBreak = errors.New("break is not in a loop or switch")

// errContinue is returned from the rendering of the "break" statement.
// It is managed by the innermost "for" statement.
var errContinue = errors.New("continue is not in a loop")

// rendering represents the state of a tree rendering.
type rendering struct {
	scope       map[string]scope
	path        string
	vars        []scope
	treeContext ast.Context
	handleError func(error) bool
}

// variables scope.
type scope map[string]interface{}

// reference represents a value in scope that has been referenced.
// rv contains the value, not the address.
type reference struct {
	rv reflect.Value
}

var scopeType = reflect.TypeOf(scope{})

// macro represents a macro in a scope.
type macro struct {
	path string
	node *ast.Macro
}

// urlState represents the rendering of rendering an URL.
type urlState struct {
	path   bool
	query  bool
	isSet  bool
	addAmp bool
}

// errorf builds and returns an rendering error.
func (r *rendering) errorf(nodeOrPos interface{}, format string, args ...interface{}) error {
	var pos *ast.Position
	if node, ok := nodeOrPos.(ast.Node); ok {
		pos = node.Pos()
		if pos == nil {
			return fmt.Errorf(format, args...)
		}
	} else {
		pos = nodeOrPos.(*ast.Position)
	}
	var err = &Error{
		Path: r.path,
		Pos: ast.Position{
			Line:   pos.Line,
			Column: pos.Column,
			Start:  pos.Start,
			End:    pos.End,
		},
		Err: fmt.Errorf(format, args...),
	}
	return err
}

func (r *rendering) renderBlock(wr io.Writer, block *ast.Block, urlstate *urlState) error {
	var err error
	r.vars = append(r.vars, nil)
	err = r.render(wr, block.Nodes, urlstate)
	r.vars = r.vars[:len(r.vars)-1]
	return err
}

// render renders nodes.
func (r *rendering) render(wr io.Writer, nodes []ast.Node, urlstate *urlState) error {

Nodes:
	for _, n := range nodes {

		switch node := n.(type) {

		case *ast.Text:

			if wr != nil {
				if len(node.Text)-node.Cut.Left-node.Cut.Right == 0 {
					continue
				}
				text := node.Text[node.Cut.Left : len(node.Text)-node.Cut.Right]
				if urlstate != nil {
					if !urlstate.query {
						if bytes.ContainsAny(text, "?#") {
							if text[0] == '?' && !urlstate.path {
								if urlstate.addAmp {
									_, err := io.WriteString(wr, "&amp;")
									if err != nil {
										return err
									}
								}
								text = text[1:]
							}
							urlstate.path = false
							urlstate.query = true
						}
						if urlstate.isSet && bytes.IndexByte(text, ',') >= 0 {
							urlstate.path = true
							urlstate.query = false
						}
					}
				}
				_, err := wr.Write(text)
				if err != nil {
					return err
				}
			}

		case *ast.URL:

			if len(node.Value) > 0 {
				isSet := node.Attribute == "srcset"
				err := r.render(wr, node.Value, &urlState{true, false, isSet, false})
				if err != nil {
					return err
				}
			}

		case *ast.Value:

			expr, err := r.eval(node.Expr)
			if err != nil {
				if r.handleError(err) {
					continue
				}
				return err
			}

			err = r.renderValue(wr, expr, node, urlstate)
			if err != nil {
				return err
			}

		case *ast.If:

			var c bool
			var err error

			r.vars = append(r.vars, nil)

			if node.Assignment != nil {
				err = r.renderAssignment(node.Assignment)
				if err != nil && !r.handleError(err) {
					return err
				}
			}
			if err == nil {
				expr, err := r.eval(node.Condition)
				if err != nil {
					if !r.handleError(err) {
						return err
					}
					expr = false
				}
				switch v := expr.(type) {
				case bool:
					c = v
				default:
					err = r.errorf(node, "non-bool %s (type %s) used as if condition", node.Condition, typeof(expr))
					if !r.handleError(err) {
						return err
					}
				}
			}
			if c {
				if node.Then != nil && len(node.Then.Nodes) > 0 {
					err = r.render(wr, node.Then.Nodes, urlstate)
					if err != nil {
						return err
					}
				}
			} else if node.Else != nil {
				var err error
				switch e := node.Else.(type) {
				case *ast.If:
					err = r.render(wr, []ast.Node{e}, urlstate)
				case *ast.Block:
					err = r.renderBlock(wr, e, urlstate)
				}
				if err != nil {
					return err
				}
			}

			r.vars = r.vars[:len(r.vars)-1]

		case *ast.For, *ast.ForRange:

			err := r.renderFor(wr, node, urlstate)
			if err != nil {
				return err
			}

		case *ast.Switch:

			// TODO (Gianluca): handle errors correctly using handleError if
			// necessary.

			var err error
			r.vars = append(r.vars, nil)

			if node.Init != nil {
				switch v := node.Init.(type) {
				case ast.Expression:
					_, err = r.eval(v)
					if err != nil && !r.handleError(err) {
						return err
					}
				case *ast.Assignment:
					err = r.renderAssignment(v)
					if err != nil && !r.handleError(err) {
						return err
					}
				}
			}

			r.vars = append(r.vars, nil)

			var conditionValue interface{}
			if node.Expr == nil {
				conditionValue = true
			} else {
				conditionValue, err = r.eval(node.Expr)
				if err != nil && !r.handleError(err) {
					return err
				}
			}

			inFallthrough := false
			var defaultCase *ast.Case
			for _, c := range node.Cases {
				// TODO (Gianluca): init render as true and invert if condition
				// removing else (if possibile)
				render := false
				isDefault := c.Expressions == nil
				if isDefault {
					defaultCase = c
				}
				if inFallthrough {
					render = true
				} else {
					for _, expr := range c.Expressions {
						res, err := r.evalBinary(conditionValue, ast.OperatorEqual, node.Expr, expr)
						if err != nil && !r.handleError(err) {
							return err
						}
						if render = res.(bool); render {
							break
						}
					}
				}
				if render {
					if inFallthrough {
						r.vars = r.vars[:len(r.vars)-1]
						r.vars = append(r.vars, nil)
					}
					err := r.render(wr, c.Body, urlstate)
					if err != nil {
						if err == errBreak {
							break
						}
						return err
					}
					if c.Fallthrough {
						inFallthrough = true
						continue
					}
					r.vars = r.vars[:len(r.vars)-2]
					defaultCase = nil
					break
				}
			}

			if defaultCase != nil {
				err := r.render(wr, defaultCase.Body, urlstate)
				if err != nil {
					return err
				}
			}

			r.vars = r.vars[:len(r.vars)-2]

		case *ast.TypeSwitch:
			// TODO (Gianluca): apply the same changes requested for case
			// *ast.Switch.

			var err error

			r.vars = append(r.vars, nil)

			if node.Init != nil {
				switch v := node.Init.(type) {
				case ast.Expression:
					_, err = r.eval(v)
					if err != nil && !r.handleError(err) {
						return err
					}
				case *ast.Assignment:
					err = r.renderAssignment(v)
					if err != nil && !r.handleError(err) {
						return err
					}
				}
			}

			ident := node.Assignment.Variables[0].(*ast.Identifier)
			expr := node.Assignment.Values[0]

			r.vars = append(r.vars, nil)

			var guardvalue interface{}

			// An assignment with a blank identifier is a type
			// assertion encapsulated within an assignment.
			if ident.Name == "_" {
				expr = node.Assignment.Values[0]
				guardvalue, err = r.eval(expr)
				if err != nil {
					return err
				}
			} else {
				err = r.renderAssignment(node.Assignment)
				if err != nil && !r.handleError(err) {
					return err
				}
				guardvalue, _ = r.variable(ident.Name)
			}
			var defaultCase *ast.Case
			for _, c := range node.Cases {
				render := false
				isDefault := c.Expressions == nil
				if isDefault {
					defaultCase = c
				} else {
					for _, expr := range c.Expressions {
						if !isDefault {
							caseExprValue, err := r.eval(expr)
							if err != nil && !r.handleError(err) {
								return err
							}
							vt2, ok := caseExprValue.(reflect.Type)
							if !ok {
								return r.errorf(c, "%s (type %s) is not a type", expr, typeof(caseExprValue))
							}
							render, err = hasType(guardvalue, vt2)
							if err != nil {
								// TODO: use r.errorf
								return err
							}
						}
						if render {
							break
						}
					}
				}
				if render {
					err := r.render(wr, c.Body, urlstate)
					if err != nil {
						if err == errBreak {
							break
						}
						return err
					}
					defaultCase = nil
					break
				}
			}

			if defaultCase != nil {
				err := r.render(wr, defaultCase.Body, urlstate)
				if err != nil {
					return err
				}
			}

			r.vars = r.vars[:len(r.vars)-2]

		case *ast.Break:
			return errBreak

		case *ast.Continue:
			return errContinue

		case *ast.Macro:
			if wr != nil {
				err := r.errorf(node.Ident, "macros not allowed")
				if !r.handleError(err) {
					return err
				}
			}
			name := node.Ident.Name
			if name == "_" {
				continue
			}
			// TODO (Gianluca): should check if referenced?
			if v, ok := r.variable(name); ok {
				var err error
				if m, ok := v.(macro); ok {
					err = r.errorf(node, "%s redeclared\n\tprevious declaration at %s:%s",
						name, m.path, m.node.Pos())
				} else {
					err = r.errorf(node.Ident, "%s redeclared in this file", name)
				}
				if r.handleError(err) {
					continue
				}
				return err
			}
			r.vars[2][name] = macro{r.path, node}

		case *ast.Assignment:

			err := r.renderAssignment(node)
			if err != nil && !r.handleError(err) {
				return err
			}

		case *ast.ShowMacro:

			name := node.Macro.Name
			if node.Import != nil {
				name = node.Import.Name + "." + name
			}
			var m macro
			var err error
			// TODO (Gianluca): should check if referenced?
			if v, ok := r.variable(name); ok {
				if m, ok = v.(macro); ok {
					if node.Context != m.node.Context {
						err = r.errorf(node, "macro %s is defined in a different context (%s)",
							name, m.node.Context)
					}
				} else {
					err = r.errorf(node, "cannot show non-macro %s (type %s)", name, typeof(v))
				}
			} else {
				err = r.errorf(node, "macro %s not declared", name)
			}
			if err != nil {
				if r.handleError(err) {
					continue
				}
				return err
			}

			isVariadic := m.node.IsVariadic

			haveSize := len(node.Arguments)
			wantSize := len(m.node.Parameters)
			if (!isVariadic && haveSize != wantSize) || (isVariadic && haveSize < wantSize-1) {
				have := "("
				for i := 0; i < haveSize; i++ {
					if i > 0 {
						have += ", "
					}
					if i < wantSize {
						have += m.node.Parameters[i].Name
					} else {
						have += "?"
					}
				}
				have += ")"
				want := "("
				for i, p := range m.node.Parameters {
					if i > 0 {
						want += ", "
					}
					want += p.Name
					if i == wantSize-1 && isVariadic {
						want += "..."
					}
				}
				want += ")"
				name := node.Macro.Name
				if node.Import != nil {
					name = node.Import.Name + " " + name
				}
				if haveSize < wantSize {
					err = r.errorf(node, "not enough arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
				} else {
					err = r.errorf(node, "too many arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
				}
				if r.handleError(err) {
					continue
				}
				return err
			}

			var last = wantSize - 1
			var arguments = scope{}
			var parameters = m.node.Parameters
			if isVariadic {
				if length := haveSize - wantSize + 1; length == 0 {
					arguments[parameters[last].Name] = []interface{}(nil)
				} else {
					arguments[parameters[last].Name] = make([]interface{}, length)
				}
			}
			for i, argument := range node.Arguments {
				arg, err := r.eval(argument)
				if err != nil {
					if r.handleError(err) {
						continue Nodes
					}
					return err
				}
				if isVariadic && i >= last {
					arguments[parameters[last].Name].([]interface{})[i-wantSize+1] = arg
				} else {
					arguments[parameters[i].Name] = arg
				}
			}
			rn := &rendering{
				scope:       r.scope,
				path:        m.path,
				vars:        []scope{r.vars[0], r.vars[1], r.scope[m.path], arguments},
				treeContext: r.treeContext,
				handleError: r.handleError,
			}
			err = rn.render(wr, m.node.Body, nil)
			if err != nil {
				return err
			}

		case *ast.Import:

			path := node.Tree.Path
			if _, ok := r.scope[path]; !ok {
				rn := &rendering{
					scope:       r.scope,
					path:        path,
					vars:        []scope{r.vars[0], r.vars[1], {}},
					treeContext: r.treeContext,
					handleError: r.handleError,
				}
				err := rn.render(nil, node.Tree.Nodes, nil)
				if err != nil {
					return err
				}
				r.scope[path] = rn.vars[2]
				if node.Ident != nil && node.Ident.Name == "_" {
					continue
				}
				for name, m := range rn.vars[2] {
					if _, ok := m.(macro); ok {
						if strings.Index(name, ".") > 0 {
							continue
						}
						if fc, _ := utf8.DecodeRuneInString(name); !unicode.Is(unicode.Lu, fc) {
							continue
						}
						r.vars[2][name] = m
					}
				}
			}

		case *ast.Include:

			r.vars = append(r.vars, nil)
			rn := &rendering{
				scope:       r.scope,
				path:        node.Tree.Path,
				vars:        r.vars,
				treeContext: r.treeContext,
				handleError: r.handleError,
			}
			err := rn.render(wr, node.Tree.Nodes, nil)
			r.vars = r.vars[:len(r.vars)-1]
			if err != nil {
				return err
			}

		case ast.Expression:

			err := r.eval0(node)
			if err != nil && !r.handleError(err) {
				return err
			}

		}
	}

	return nil
}

// variable returns the value of the variable name in rendering r.
func (r *rendering) variable(name string) (interface{}, bool) {
	for i := len(r.vars) - 1; i >= 0; i-- {
		if r.vars[i] != nil {
			if v, ok := r.vars[i][name]; ok {
				switch v.(type) {
				case reference:
					panic("referenced not implemented in (*rendering).variable")
				default:
					return v, true
				}

			}
		}
	}
	return nil, false
}

// typeof returns the string representation of the type of v.
// If v is nil returns "nil".
// TODO (Gianluca): to review.
func typeof(v interface{}) string {
	switch vv := v.(type) {
	case nil:
		return "nil"
	case HTML:
		return "string"
	case Map:
		return "map"
	case Slice:
		return "slice"
	case Bytes:
		return "bytes"
	case ConstantNumber:
		return "untyped number"
	case CustomNumber:
		return vv.Name()
	}
	return fmt.Sprintf("%T", v)
}
