// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"go/constant"
	gotoken "go/token"
	"math/big"
	"reflect"

	"scrigo/ast"
)

const noEllipses = -1

type builtinfunc struct{}

var intType = reflect.TypeOf(0)
var uint8Type = reflect.TypeOf(uint8(0))
var builtinfuncType = reflect.TypeOf(builtinfunc{})

var numericKind = [...]bool{
	reflect.Int:     true,
	reflect.Int8:    true,
	reflect.Int16:   true,
	reflect.Int32:   true,
	reflect.Int64:   true,
	reflect.Uint:    true,
	reflect.Uint8:   true,
	reflect.Uint16:  true,
	reflect.Uint32:  true,
	reflect.Uint64:  true,
	reflect.Float32: true,
	reflect.Float64: true,
}

var boolOperators = [15]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
	ast.OperatorAnd:      true,
	ast.OperatorOr:       true,
}

var intOperators = [15]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
	ast.OperatorModulo:         true,
}

var floatOperators = [15]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
}

var stringOperators = [15]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
}

var operatorsOfKind = [...][15]bool{
	reflect.Bool:    boolOperators,
	reflect.Int:     intOperators,
	reflect.Int8:    intOperators,
	reflect.Int16:   intOperators,
	reflect.Int32:   intOperators,
	reflect.Int64:   intOperators,
	reflect.Uint:    intOperators,
	reflect.Uint8:   intOperators,
	reflect.Uint16:  intOperators,
	reflect.Uint32:  intOperators,
	reflect.Uint64:  intOperators,
	reflect.Float32: floatOperators,
	reflect.Float64: floatOperators,
	reflect.String:  stringOperators,
}

// typechecker represents the state of a type checking.
type typechecker struct {
	path string
}

// errorf builds and returns a type check error.
func (tc *typechecker) errorf(nodeOrPos interface{}, format string, args ...interface{}) error {
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
		Path: tc.path,
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

// checkExpression returns the type info of expr. Returns an error if expr is
// a type or a package.
func (tc *typechecker) checkExpression(expr ast.Expression) *ast.TypeInfo {
	ti := tc.typeof(expr, noEllipses)
	if ti.IsType() || ti.IsPackage() {
		panic(tc.errorf(expr, "%s is not an expression", ti))
	}
	expr.SetTypeInfo(ti)
	return ti
}

// checkType evaluates expr as a type and returns the type info. Returns an
// error if expr is not an type.
func (tc *typechecker) checkType(expr ast.Expression, length int) *ast.TypeInfo {
	ti := tc.typeof(expr, length)
	if !ti.IsType() {
		panic(tc.errorf(expr, "%s is not a type", ti))
	}
	expr.SetTypeInfo(ti)
	return ti
}

// typeof returns the type of expr. If expr is not an expression but a type,
// returns the type.
func (tc *typechecker) typeof(expr ast.Expression, length int) *ast.TypeInfo {

	switch expr := expr.(type) {

	case *ast.String:
		return &ast.TypeInfo{
			Constant: &ast.Constant{
				DefaultType: ast.DefaultTypeString,
				String:      expr.Text,
			},
		}

	case *ast.Int:
		return &ast.TypeInfo{
			Constant: &ast.Constant{
				DefaultType: ast.DefaultTypeInt,
				Number:      constant.MakeFromLiteral(expr.Value.String(), gotoken.INT, 0),
			},
		}

	case *ast.Rune:
		return &ast.TypeInfo{
			Constant: &ast.Constant{
				DefaultType: ast.DefaultTypeRune,
				Number:      constant.MakeInt64(int64(expr.Value)),
			},
		}

	case *ast.Float:
		return &ast.TypeInfo{
			Constant: &ast.Constant{
				DefaultType: ast.DefaultTypeFloat64,
				Number:      constant.MakeFromLiteral(expr.Value.Text('f', -1), gotoken.FLOAT, 0),
			},
		}

	case *ast.Parentesis:
		panic("unexpected parentesis")

	case *ast.UnaryOperator:
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "invalid operation: ! nil"))
		}
		switch expr.Op {
		case ast.OperatorNot:
			if c := t.Constant; c != nil {
				if t.Constant.DefaultType != ast.DefaultTypeBool {
					panic(tc.errorf(expr, "invalid operation: ! %s", expr))
				}
				c = &ast.Constant{DefaultType: c.DefaultType, Bool: !c.Bool}
				t = &ast.TypeInfo{Properties: t.Properties, Type: t.Type, Constant: c}
			} else if t.Type != nil && t.Type.Kind() != reflect.Bool {
				panic(tc.errorf(expr, "invalid operation: ! %s", expr))
			}
			return t
		case ast.OperatorAddition, ast.OperatorSubtraction:
			if c := t.Constant; c != nil {
				if c.DefaultType == ast.DefaultTypeString || c.DefaultType == ast.DefaultTypeBool {
					panic(tc.errorf(expr, "invalid operation: %s %s", expr.Op, t))
				}
				if expr.Op == ast.OperatorSubtraction {
					c = &ast.Constant{DefaultType: c.DefaultType, Number: constant.UnaryOp(gotoken.SUB, c.Number, 0)}
					t = &ast.TypeInfo{Properties: t.Properties, Type: t.Type, Constant: c}
				}
			} else if t.Type != nil && !numericKind[t.Type.Kind()] {
				panic(tc.errorf(expr, "invalid operation: %s %s", expr.Op, t))
			}
			return t
		}

	case *ast.BinaryOperator:
		t1 := tc.checkExpression(expr.Expr1)
		t2 := tc.checkExpression(expr.Expr2)
		if t1.Nil() && t2.Nil() {
			panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on nil)", expr, expr.Op))
		}
		if t1.Nil() || t2.Nil() {
			var t = t1
			if t.Nil() {
				t = t2
			}
			if t.Type != nil && !t.Type.Comparable() {
				panic(tc.errorf(expr, "cannot convert nil to type %s", t))
			}
			if expr.Op != ast.OperatorEqual && expr.Op != ast.OperatorNotEqual {
				panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t.Type.Kind()))
			}
			return &ast.TypeInfo{}
		}
		if t1.Constant != nil && t2.Constant != nil {
			mismatch := false
			df2 := t2.Constant.DefaultType
			switch t1.Constant.DefaultType {
			default:
				mismatch = df2 == ast.DefaultTypeString || df2 == ast.DefaultTypeBool
			case ast.DefaultTypeString:
				mismatch = df2 != ast.DefaultTypeString
			case ast.DefaultTypeBool:
				mismatch = df2 != ast.DefaultTypeBool
			}
			if mismatch {
				panic(tc.errorf(expr, "invalid operation: %v (mismatched types %s and %s)", expr, t1, t2))
			}
			return tc.binaryOp(expr)
		}
		if t1.Type == nil && t2.Type == nil {
			t2 = tc.convert(expr.Expr2, t1.Type)
		}
		if t1.Type == nil {
			t1 = tc.convert(expr.Expr1, t2.Type)
		} else if t1.Type == nil {
			t2 = tc.convert(expr.Expr2, t1.Type)
		}
		if expr.Op >= ast.OperatorEqual && expr.Op <= ast.OperatorGreaterOrEqual {
			if !tc.isAssignableTo(t1, t2.Type) && !tc.isAssignableTo(t2, t1.Type) {
				panic(tc.errorf(expr, "invalid operation: %v (mismatched types %s and %s)", expr, t1, t2))
			}
			if expr.Op == ast.OperatorEqual || expr.Op == ast.OperatorNotEqual {
				if !tc.isComparable(t1) {
					// TODO(marco) explain in the error message why they are not comparable.
					panic(tc.errorf(expr, "invalid operation: %v (%s cannot be compared)", expr, t1.Type))
				}
			} else if !tc.isOrdered(t1) {
				panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1.Type.Kind()))
			}
			return &ast.TypeInfo{}
		}
		if t1.Type != t2.Type {
			panic(tc.errorf(expr, "invalid operation: %v (mismatched types %s and %s)", expr, t1, t2))
		}
		if kind := t1.Type.Kind(); !operatorsOfKind[kind][expr.Op] {
			panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, kind))
		}
		return t1

	case *ast.Identifier:
		// TODO
		//if typ.IsPackage() {
		//	return nil, tc.errorf(expr, "use of package %s without selector", typ)
		//}

	case *ast.MapType:
		key := tc.checkType(expr.KeyType, noEllipses)
		value := tc.checkType(expr.ValueType, noEllipses)
		defer func() {
			if rec := recover(); rec != nil {
				panic(tc.errorf(expr, "invalid map key type %s", key))
			}
		}()
		return &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.MapOf(key.Type, value.Type)}

	case *ast.SliceType:
		elem := tc.checkType(expr.ElementType, noEllipses)
		return &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.SliceOf(elem.Type)}

	case *ast.CompositeLiteral:

	case *ast.Func:
		variadic := expr.Type.IsVariadic
		// Parameters.
		numIn := len(expr.Type.Parameters)
		in := make([]reflect.Type, numIn)
		for i := numIn - 1; i >= 0; i-- {
			param := expr.Type.Parameters[i]
			if param.Type == nil {
				in[i] = in[i+1]
			} else {
				t := tc.checkType(param.Type, noEllipses)
				if variadic && i == numIn-1 {
					in[i] = reflect.SliceOf(t.Type)
				} else {
					in[i] = t.Type
				}
			}
		}
		// Result.
		numOut := len(expr.Type.Result)
		out := make([]reflect.Type, numOut)
		for i := numOut - 1; i >= 0; i-- {
			res := expr.Type.Result[i]
			if res.Type == nil {
				out[i] = out[i+1]
			} else {
				c := tc.checkType(res.Type, noEllipses)
				out[i] = c.Type
			}
		}
		return &ast.TypeInfo{Type: reflect.FuncOf(in, out, variadic)}

	case *ast.Call:
		types := tc.checkCallExpression(expr, false)
		if len(types) == 0 {
			panic(tc.errorf(expr, "%v used as value", expr))
		}
		if len(types) > 1 {
			panic(tc.errorf(expr, "multiple-value %v in single-value context", expr))
		}

	case *ast.Index:
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		switch kind := t.Type.Kind(); kind {
		case reflect.Slice, reflect.String, reflect.Array, reflect.Ptr:
			if kind == reflect.Ptr && t.Type.Elem().Kind() != reflect.Array {
				panic(tc.errorf(expr, "invalid operation: %v (type %s does not support indexing)", expr, t))
			}
			index := tc.checkExpression(expr.Index)
			if index.Nil() || index.Type != intType {
				k := kind
				if kind == reflect.Ptr {
					k = reflect.Array
				}
				if index == nil {
					panic(tc.errorf(expr, "non-integer %s index nil", kind))
				}
				panic(tc.errorf(expr, "non-integer %s index %s", k, index))
			}
			switch kind {
			case reflect.String:
				return &ast.TypeInfo{Type: uint8Type}
			case reflect.Slice, reflect.Array:
				return &ast.TypeInfo{Type: t.Type.Elem()}
			case reflect.Ptr:
				return &ast.TypeInfo{Type: t.Type.Elem().Elem()}
			}
		case reflect.Map:
			key := tc.checkExpression(expr.Index)
			if !tc.isAssignableTo(key, t.Type.Key()) {
				if key.Nil() {
					panic(tc.errorf(expr, "cannot convert nil to type %s", t.Type.Key()))
				}
				panic(tc.errorf(expr, "cannot use %v (type %s) as type %s in map index", expr.Index, key, t.Type.Key()))
			}
			return &ast.TypeInfo{Type: t.Type.Elem()}

		default:
			panic(tc.errorf(expr, "invalid operation: %v (type %s does not support indexing)", expr, t))
		}

	case *ast.Slicing:
		// TODO(marco) support full slice expressions
		// TODO(marco) check if an array is addressable
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		kind := t.Type.Kind()
		switch kind {
		case reflect.String, reflect.Slice, reflect.Array:
		default:
			if kind != reflect.Ptr || t.Type.Elem().Kind() != reflect.Array {
				panic(tc.errorf(expr, "cannot slice %v (type %s)", expr.Expr, t))
			}
		}
		if expr.Low != nil {
			low := tc.checkExpression(expr.Low)
			if low.Nil() {
				panic(tc.errorf(expr, "invalid slice index nil (type nil)"))
			}
			if low.Type.Kind() != reflect.Int {
				panic(tc.errorf(expr, "invalid slice index %v (type %s)", expr.Low, low))
			}
		}
		if expr.High != nil {
			high := tc.checkExpression(expr.High)
			if high.Nil() {
				panic(tc.errorf(expr, "invalid slice index nil (type nil)"))
			}
			if high.Type.Kind() != reflect.Int {
				panic(tc.errorf(expr, "invalid slice index %v (type %s)", expr.High, high))
			}
		}
		switch kind {
		case reflect.String, reflect.Slice:
			return t
		case reflect.Array:
			return &ast.TypeInfo{Type: reflect.SliceOf(t.Type.Elem())}
		case reflect.Ptr:
			return &ast.TypeInfo{Type: reflect.SliceOf(t.Type.Elem().Elem())}
		}

	case *ast.Selector:
		if ident, ok := expr.Expr.(*ast.Identifier); ok {
			var t = ident.TypeInfo()
			if t.IsPackage() {
				// TODO(marco)
			}
			if t.IsType() {
				method, ok := tc.methodByName(t, expr.Ident)
				if !ok {
					panic(tc.errorf(expr, "%v undefined (type %s has no method %s)", expr, t, expr.Ident))
				}
				return method
			}
		}
		t := tc.checkExpression(expr.Expr)
		method, ok := tc.methodByName(t, expr.Ident)
		if ok {
			return method
		}
		field, ok := tc.fieldByName(t, expr.Ident)
		if ok {
			return field
		}
		panic(tc.errorf(expr, "%v undefined (type %s has no field or method %s)", expr, t, expr.Ident))

	case *ast.TypeAssertion:
		t := tc.typeof(expr.Expr, noEllipses)
		if t.Type.Kind() != reflect.Interface {
			panic(tc.errorf(expr, "invalid type assertion: %v (non-interface type %s on left)", expr, t))
		}
		expr.Expr.SetTypeInfo(t)
		if expr.Type == nil {
			return nil
		}
		t = tc.checkType(expr.Type, noEllipses)
		expr.Type.SetTypeInfo(t)
		return t

	}

	panic(tc.errorf(expr, "unexpected node"))
}

// checkCallExpression type checks a call expression, including type
// conversions and built-in function calls.
func (tc *typechecker) checkCallExpression(expr *ast.Call, statement bool) []*ast.TypeInfo {

	t := tc.typeof(expr.Func, noEllipses)

	if t.Nil() {
		panic(tc.errorf(expr, "use of untyped nil"))
	}

	if t.IsType() {
		if len(expr.Args) == 0 {
			panic(tc.errorf(expr, "missing argument to conversion to %s: %s", t, expr))
		}
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to conversion to %s: %s", t, expr))
		}
		arg := tc.checkExpression(expr.Args[0])
		if !arg.Type.ConvertibleTo(t.Type) {
			panic(tc.errorf(expr, "cannot convert %v (type %s) to type %s", expr.Args[0], arg, t))
		}
		return []*ast.TypeInfo{t}
	}

	if t.IsPackage() {
		panic(tc.errorf(expr, "use of package fmt without selector"))
	}

	if t.Type == builtinfuncType {

		ident := expr.Func.(*ast.Identifier)

		switch ident.Name {

		case "append":
			if len(expr.Args) == 0 {
				panic(tc.errorf(expr, "missing arguments to append"))
			}
			slice := tc.checkExpression(expr.Args[0])
			if slice.Nil() {
				panic(tc.errorf(expr, "first argument to append must be typed slice; have untyped nil"))
			}
			if slice.Type.Kind() != reflect.Slice {
				panic(tc.errorf(expr, "first argument to append must be slice; have %s", t))
			}
			if len(expr.Args) > 1 {
				// TODO(marco): implements variadic call to append
				// TODO(marco): implements append([]byte{}, "abc"...)
				elem := t.Type.Elem()
				for _, el := range expr.Args[1:] {
					t := tc.checkExpression(el)
					if !tc.isAssignableTo(t, elem) {
						if t == nil {
							panic(tc.errorf(expr, "cannot use nil as type %s in append", elem))
						}
						panic(tc.errorf(expr, "cannot use %v (type %s) as type %s in append", el, t, elem))
					}
				}
			}
			return []*ast.TypeInfo{slice}

		case "copy":
			if len(expr.Args) < 2 {
				panic(tc.errorf(expr, "missing argument to copy: %s", expr))
			}
			if len(expr.Args) > 2 {
				panic(tc.errorf(expr, "too many arguments to copy: %s", expr))
			}
			dst := tc.checkExpression(expr.Args[0])
			src := tc.checkExpression(expr.Args[1])
			if dst.Nil() || src.Nil() {
				panic(tc.errorf(expr, "use of untyped nil"))
			}
			dk := dst.Type.Kind()
			sk := dst.Type.Kind()
			switch {
			case dk != reflect.Slice && sk != reflect.Slice:
				panic(tc.errorf(expr, "arguments to copy must be slices; have %s, %s", dst, src))
			case dk != reflect.Slice:
				panic(tc.errorf(expr, "first argument to copy should be slice; have %s", dst))
			case sk != reflect.Slice && sk != reflect.String:
				panic(tc.errorf(expr, "second argument to copy should be slice or string; have %s", src))
			case sk == reflect.String && dst.Type.Elem().Kind() != reflect.Uint8:
				panic(tc.errorf(expr, "arguments to copy have different element types: %s and %s", dst, src))
			}
			// TODO(marco): verificare se il confronto dei reflect.typ è sufficiente per essere conformi alle specifiche
			if dk == reflect.Slice && sk == reflect.Slice && dst != src {
				panic(tc.errorf(expr, "arguments to copy have different element types: %s and %s", dst, src))
			}
			return []*ast.TypeInfo{{Type: intType}}

		case "delete":
			if len(expr.Args) < 2 {
				panic(tc.errorf(expr, "missing argument to delete: %s", expr))
			}
			if len(expr.Args) > 2 {
				panic(tc.errorf(expr, "too many arguments to delete: %s", expr))
			}
			t := tc.checkExpression(expr.Args[0])
			key := tc.checkExpression(expr.Args[1])
			if t.Nil() {
				panic(tc.errorf(expr, "first argument to delete must be map; have nil"))
			}
			if t.Type.Kind() != reflect.Map {
				panic(tc.errorf(expr, "first argument to delete must be map; have %s", t))
			}
			if rkey := t.Type.Key(); !tc.isAssignableTo(key, rkey) {
				if key == nil {
					panic(tc.errorf(expr, "cannot use nil as type %s in delete", rkey))
				}
				panic(tc.errorf(expr, "cannot use %v (type %s) as type %s in delete", expr.Args[1], key, rkey))
			}
			return nil

		case "len":
			if len(expr.Args) < 1 {
				panic(tc.errorf(expr, "missing argument to len: %s", expr))
			}
			if len(expr.Args) > 1 {
				panic(tc.errorf(expr, "too many arguments to len: %s", expr))
			}
			t := tc.checkExpression(expr.Args[0])
			if t.Nil() {
				panic(tc.errorf(expr, "use of untyped nil"))
			}
			switch k := t.Type.Kind(); k {
			case reflect.String, reflect.Array, reflect.Map, reflect.Chan:
			default:
				if k != reflect.Ptr || t.Type.Elem().Kind() != reflect.Array {
					panic(tc.errorf(expr, "invalid argument %v (type %s) for len", expr, t))
				}
			}
			return []*ast.TypeInfo{{Type: intType}}

		case "make":
			numArgs := len(expr.Args)
			if numArgs == 0 {
				panic(tc.errorf(expr, "missing argument to make"))
			}
			c := tc.typeof(expr.Args[0], noEllipses)
			if !c.IsType() {
				panic(tc.errorf(expr.Args[0], "%s is not a type", expr.Args[0]))
			}
			switch c.Type.Kind() {
			case reflect.Slice:
				if numArgs == 1 {
					panic(tc.errorf(expr, "missing len argument to make(%s)", expr.Args[0]))
				}
				if numArgs > 1 {
					len := tc.checkExpression(expr.Args[1])
					if len == nil {
						panic(tc.errorf(expr, "non-integer len argument in make(%s) - nil", expr.Args[0]))
					}
					if len.Type != intType {
						panic(tc.errorf(expr, "non-integer len argument in make(%s) - %s", expr.Args[0], len))
					}
				}
				if numArgs > 2 {
					cap := tc.checkExpression(expr.Args[2])
					if cap.Nil() {
						panic(tc.errorf(expr, "non-integer cap argument in make(%s) - nil", expr.Args[0]))
					}
					if cap.Type != intType {
						panic(tc.errorf(expr, "non-integer cap argument in make(%s) - %s", expr.Args[0], cap))
					}
				}
				if numArgs > 3 {
					panic(tc.errorf(expr, "too many arguments to make(%s)", expr.Args[0]))
				}
			case reflect.Map:
				if numArgs > 2 {
					panic(tc.errorf(expr, "too many arguments to make(%s)", expr.Args[0]))
				}
				if numArgs == 2 {
					len := tc.checkExpression(expr.Args[1])
					if len.Nil() {
						panic(tc.errorf(expr, "cannot convert nil to type int"))
					}
					if len.Type != intType {
						panic(tc.errorf(expr, "non-integer size argument in make(%s) - %s", expr.Args[0], len))
					}
				}
			default:
				panic(tc.errorf(expr, "cannot make type %s", c))
			}
			return []*ast.TypeInfo{t}

		case "new":
			if len(expr.Args) == 0 {
				panic(tc.errorf(expr, "missing argument to new"))
			}
			t := tc.typeof(expr.Args[0], noEllipses)
			if t.IsType() {
				panic(tc.errorf(expr, "%s is not a type", expr.Args[0]))
			}
			if len(expr.Args) > 1 {
				panic(tc.errorf(expr, "too many arguments to new(%s)", expr.Args[0]))
			}
			return []*ast.TypeInfo{{Type: reflect.PtrTo(t.Type)}}

		case "panic":
			if len(expr.Args) == 0 {
				panic(tc.errorf(expr, "missing argument to panic: panic()"))
			}
			if len(expr.Args) > 1 {
				panic(tc.errorf(expr, "too many arguments to panic: %s", expr))
			}
			_ = tc.checkExpression(expr.Args[0])
			return nil

		}

		panic(fmt.Sprintf("unexpected builtin %s", ident.Name))

	}

	if t.Type.Kind() != reflect.Func {
		panic(tc.errorf(expr, "cannot call non-function %v (type %s)", expr.Func, t))
	}

	var numIn = t.Type.NumIn()
	var variadic = t.Type.IsVariadic()

	if (!variadic && len(expr.Args) != numIn) || (variadic && len(expr.Args) < numIn-1) {
		have := "("
		for i, arg := range expr.Args {
			if i > 0 {
				have += ", "
			}
			c := tc.checkExpression(arg)
			if c == nil {
				have += "nil"
			} else {
				have += t.String()
			}
		}
		have += ")"
		want := "("
		for i := 0; i < numIn; i++ {
			if i > 0 {
				want += ", "
			}
			if i == numIn-1 && variadic {
				want += "..."
			}
			want += t.Type.In(i).String()
		}
		want += ")"
		if len(expr.Args) < numIn {
			panic(tc.errorf(expr, "not enough arguments in call to %s\n\thave %s\n\twant %s", expr.Func, have, want))
		}
		panic(tc.errorf(expr, "too many arguments in call to %s\n\thave %s\n\twant %s", expr.Func, have, want))
	}

	var lastIn = numIn - 1
	var in reflect.Type

	for i, arg := range expr.Args {
		if i < lastIn || !variadic {
			in = t.Type.In(i)
		} else if i == lastIn {
			in = t.Type.In(lastIn).Elem()
		}
		a := tc.checkExpression(arg)
		if !tc.isAssignableTo(a, in) {
			panic(tc.errorf(expr.Args[i], "cannot use %s (type %s) as type %s in argument to %s", expr.Args[i], a, in, expr.Func))
		}
	}

	numOut := t.Type.NumOut()
	resultTypes := make([]*ast.TypeInfo, numOut)
	for i := 0; i < numOut; i++ {
		resultTypes[i] = &ast.TypeInfo{Type: t.Type.Out(i)}
	}

	return resultTypes
}

// binaryOp executes the binary expression c op u, where c and u are constants.
// The tow operands must be both numeric, boolean or string.
// Panics if it can not be executed.
func (tc *typechecker) binaryOp(expr *ast.BinaryOperator) *ast.TypeInfo {
	c := ast.Constant{}
	c1 := expr.Expr1.TypeInfo().Constant
	c2 := expr.Expr2.TypeInfo().Constant
	switch c1.DefaultType {
	default:
		var v interface{}
		switch expr.Op {
		case ast.OperatorEqual:
			v = constant.Compare(c1.Number, gotoken.EQL, c2.Number)
		case ast.OperatorNotEqual:
			v = constant.Compare(c1.Number, gotoken.NEQ, c2.Number)
		case ast.OperatorLess:
			v = constant.Compare(c1.Number, gotoken.LSS, c2.Number)
		case ast.OperatorLessOrEqual:
			v = constant.Compare(c1.Number, gotoken.LEQ, c2.Number)
		case ast.OperatorGreater:
			v = constant.Compare(c1.Number, gotoken.GTR, c2.Number)
		case ast.OperatorGreaterOrEqual:
			v = constant.Compare(c1.Number, gotoken.GEQ, c2.Number)
		case ast.OperatorAddition:
			v = constant.BinaryOp(c1.Number, gotoken.ADD, c2.Number)
		case ast.OperatorSubtraction:
			v = constant.BinaryOp(c1.Number, gotoken.SUB, c2.Number)
		case ast.OperatorMultiplication:
			v = constant.BinaryOp(c1.Number, gotoken.MUL, c2.Number)
		case ast.OperatorDivision:
			if constant.Sign(c2.Number) == 0 {
				panic(errDivisionByZero)
			}
			if c1.DefaultType == ast.DefaultTypeFloat64 || c2.DefaultType == ast.DefaultTypeFloat64 {
				v = constant.BinaryOp(c1.Number, gotoken.QUO, c2.Number)
			} else {
				a, _ := new(big.Int).SetString(c1.Number.ExactString(), 10)
				b, _ := new(big.Int).SetString(c2.Number.ExactString(), 10)
				v = constant.MakeFromLiteral(a.Quo(a, b).String(), gotoken.INT, 0)
			}
		case ast.OperatorModulo:
			if c1.DefaultType == ast.DefaultTypeFloat64 || c2.DefaultType == ast.DefaultTypeFloat64 {
				panic(errFloatModulo)
			}
			if constant.Sign(c2.Number) == 0 {
				panic(errDivisionByZero)
			}
			v = constant.BinaryOp(c1.Number, gotoken.REM, c2.Number)
		}
		if number, ok := v.(constant.Value); ok {
			c.DefaultType = c1.DefaultType
			if c.DefaultType < c2.DefaultType {
				c.DefaultType = c2.DefaultType
			}
			c.Number = number
		} else {
			c.DefaultType = ast.DefaultTypeBool
			c.Bool = v.(bool)
		}
	case ast.DefaultTypeString:
		switch expr.Op {
		case ast.OperatorEqual, ast.OperatorNotEqual:
			c.DefaultType = ast.DefaultTypeBool
			c.Bool = c1.Bool == c2.Bool
			if expr.Op == ast.OperatorNotEqual {
				c.Bool = !c.Bool
			}
		case ast.OperatorAddition:
			c.DefaultType = ast.DefaultTypeString
			c.String = c1.String + c2.String
		}
	case ast.DefaultTypeBool:
		c.DefaultType = ast.DefaultTypeBool
		switch expr.Op {
		case ast.OperatorEqual:
			c.Bool = c1.Bool == c2.Bool
		case ast.OperatorNotEqual:
			c.Bool = c1.Bool != c2.Bool
		case ast.OperatorAnd:
			c.Bool = c1.Bool && c2.Bool
		case ast.OperatorOr:
			c.Bool = c1.Bool || c2.Bool
		default:
			panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, c1.DefaultType))
		}
	}
	return &ast.TypeInfo{Constant: &c}
}

// convert converts the untyped value (constant or not constant) of expr to
// type rt or to the default value if rt is nil. After the conversion c is a
// typed value. Panics if it can not be converted.
func (tc *typechecker) convert(expr ast.Expression, rt reflect.Type) *ast.TypeInfo {
	ti := expr.TypeInfo()
	if ti.Type != nil {
		panic("convert on a typed value")
	}
	if ti.Constant == nil {
		// expr is an untyped not constant bool.
		if rt.Kind() != reflect.Bool {
			panic(fmt.Errorf("cannot convert %s (type untyped bool) to type %s", expr, rt))
		}
		return &ast.TypeInfo{Type: rt}
	}
	// t is an untyped constant.
	switch ti.Constant.DefaultType {
	case ast.DefaultTypeBool:
		if rt.Kind() != reflect.Bool {
			panic(fmt.Errorf("cannot convert %s (type untyped bool) to type %s", expr, rt))
		}
	case ast.DefaultTypeString:
		if rt.Kind() != reflect.String {
			panic(fmt.Errorf("cannot convert %s (type untyped string) to type %s", expr, rt))
		}
	default:
		cn := ConstantNumber{val: ti.Constant.Number, typ: ConstantNumberType(ti.Constant.DefaultType)}
		_, err := cn.ToType(rt)
		if err != nil {
			if _, ok := err.(errConstantConversion); ok {
				panic(fmt.Errorf("cannot convert %s (type untyped number) to type %s", expr, rt))
			}
			panic(err)
		}
	}
	return &ast.TypeInfo{Type: rt}
}

// fieldByName returns the struct field with the given name and a boolean
// indicating if the field was found.
func (tc *typechecker) fieldByName(t *ast.TypeInfo, name string) (*ast.TypeInfo, bool) {
	field, ok := t.Type.FieldByName(name)
	if ok {
		return &ast.TypeInfo{Type: field.Type}, true
	}
	if t.Type.Kind() == reflect.Ptr {
		field, ok = t.Type.Elem().FieldByName(name)
		if ok {
			return &ast.TypeInfo{Type: field.Type}, true
		}
	}
	return nil, false
}

// isAssignableTo reports whether t1 is assignable to type t2.
func (tc *typechecker) isAssignableTo(t1 *ast.TypeInfo, t2 reflect.Type) bool {
	if t1.Nil() {
		switch t2.Kind() {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return true
		}
		return false
	}
	return t1.Type.AssignableTo(t2)
}

// isComparable reports whether t is comparable.
func (tc *typechecker) isComparable(t *ast.TypeInfo) bool {
	if t.Type == nil {
		return true
	}
	return t.Type.Comparable()
}

// isOrdered reports whether t is ordered.
func (tc *typechecker) isOrdered(t *ast.TypeInfo) bool {
	if t.Type == nil {
		// Untyped bool values (constant or not constant) are not ordered.
		return t.Constant != nil && t.Constant.DefaultType != ast.DefaultTypeBool
	}
	k := t.Type.Kind()
	return numericKind[k] || k == reflect.String
}

// methodByName returns a function type that describe the method with that
// name and a boolean indicating if the method was found.
//
// Only for type classes, the returned function type has the method's
// receiver as first argument.
func (tc *typechecker) methodByName(t *ast.TypeInfo, name string) (*ast.TypeInfo, bool) {
	if t.IsType() {
		if method, ok := t.Type.MethodByName(name); ok {
			return &ast.TypeInfo{Type: method.Type}, true
		}
		return nil, false
	}
	method := reflect.Zero(t.Type).MethodByName(name)
	if method.IsValid() {
		return &ast.TypeInfo{Type: method.Type()}, true
	}
	if t.Type.Kind() != reflect.Ptr {
		method = reflect.Zero(reflect.PtrTo(t.Type)).MethodByName(name)
		if method.IsValid() {
			return &ast.TypeInfo{Type: method.Type()}, true
		}
	}
	return nil, false
}
