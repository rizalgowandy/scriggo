// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"reflect"

	"scrigo/ast"
)

// checkNodesInNewScope checks nodes in a dedicated scope, which will be
// destroyed after use.
func (tc *typechecker) checkNodesInNewScope(nodes []ast.Node) {
	tc.addScope()
	tc.checkNodes(nodes)
	tc.removeCurrentScope()
}

// checkNodes checks nodes an orderer list of statements.
//
// TODO (Gianluca): check if !nil before calling 'tc.checkNodes' and
// 'tc.checkNodesInNewScope'
//
func (tc *typechecker) checkNodes(nodes []ast.Node) {

	tc.terminating = false

	for _, node := range nodes {

		switch node := node.(type) {

		case *ast.Extends:

			panic("found *ast.Extends") // TODO (Gianluca): to review.

		case *ast.Include:

			tc.checkNodesInNewScope(node.Tree.Nodes)

		case *ast.Block:

			tc.checkNodesInNewScope(node.Nodes)

		case *ast.If:

			terminating := true
			tc.addScope()
			if node.Assignment != nil {
				tc.checkAssignment(node.Assignment)
			}
			expr := tc.checkExpression(node.Condition)
			// TODO(marco): types with underlying type bool and the untyped bool are both allowed as condition.
			// TODO (Gianluca): currently using isAssignableTo (not sure if it's right)
			// if expr.Type != boolType {
			// 	panic(tc.errorf(node.Condition, "non-bool %v (type %s) used as if condition", node.Condition, expr.Type))
			// }
			if !isAssignableTo(expr, boolType) {
				// TODO (Gianluca): error message must include default type.
				panic(tc.errorf(node.Condition, "non-bool %s (type %v) used as if condition", node.Condition, expr.ShortString()))
			}
			if node.Then != nil {
				tc.checkNodesInNewScope(node.Then.Nodes)
				terminating = terminating && tc.terminating
			}
			if node.Else != nil {
				switch els := node.Else.(type) {
				case *ast.Block:
					tc.checkNodesInNewScope(els.Nodes)
				case *ast.If:
					// TODO (Gianluca): same problem we had in renderer:
					tc.checkNodes([]ast.Node{els})
				}
				terminating = terminating && tc.terminating
			} else {
				terminating = false
			}
			tc.removeCurrentScope()
			tc.terminating = terminating

		case *ast.For:

			terminating := true
			tc.addScope()
			tc.addToAncestors(node)
			if node.Init != nil {
				nVars := len(node.Init.Variables)
				nValues := len(node.Init.Values)
				if nVars == 2 && nValues == 1 {
					intTypeInfo := &TypeInfo{Type: reflect.TypeOf(int(0))} // TODO (Gianluca): to review.
					isDecl := node.Init.Type == ast.AssignmentDeclaration
					tc.assignSingle(node.Init, node.Init.Variables[0], nil, intTypeInfo, nil, isDecl, false)
					elemTi := tc.checkExpression(node.Init.Values[0])
					tc.assignSingle(node.Init, node.Init.Variables[1], nil, &TypeInfo{Type: elemTi.Type.Elem()}, nil, isDecl, false)
				} else {
					tc.checkAssignment(node.Init)
				}
			}
			if node.Condition != nil {
				terminating = false
				expr := tc.checkExpression(node.Condition)
				// TODO (Gianluca): same as for if
				if !isAssignableTo(expr, boolType) {
					// TODO (Gianluca): error message must include default type.
					panic(tc.errorf(node.Condition, "non-bool %s (type %v) used as for condition", node.Condition, expr.ShortString()))
				}
			}
			if node.Post != nil {
				tc.checkAssignment(node.Post)
			}
			// TODO (Gianluca): can node.Body be nil?
			tc.checkNodesInNewScope(node.Body)
			tc.removeLastAncestor()
			tc.removeCurrentScope()
			tc.terminating = terminating && !tc.hasBreak[node]

		case *ast.ForRange:

			tc.addScope()
			tc.addToAncestors(node)
			if node.Assignment != nil {
				if len(node.Assignment.Variables) > 2 {
					panic(tc.errorf(node, "too many variables in range"))
				}
				rangeExpr := tc.checkExpression(node.Assignment.Values[0])
				var key, elem reflect.Type
				switch typ := rangeExpr.Type; typ.Kind() {
				case reflect.Array, reflect.Slice:
					key = intType
					elem = typ.Elem()
				case reflect.Map:
					key = typ.Key()
					elem = typ.Elem()
				case reflect.String:
					key = intType
					elem = reflect.TypeOf(rune(' '))
				case reflect.Ptr:
					if typ.Elem().Kind() != reflect.Array {
						panic(tc.errorf(node.Assignment.Values[0], "cannot range over %s (type %s)", node.Assignment.Values[0], rangeExpr.String()))
					}
					key = intType
					elem = typ.Elem().Elem()
				case reflect.Chan:
					if typ.ChanDir() == reflect.RecvDir {
						panic(tc.errorf(node.Assignment.Values[0], "invalid operation: range %s (receive from send-only type %s)", node.Assignment.Values[0], rangeExpr.String()))
					}
					if len(node.Assignment.Variables) == 2 {
						panic(tc.errorf(node, "too many variables in range"))
					}
					elem = typ.Elem()
				default:
					panic(tc.errorf(node.Assignment.Values[0], "cannot range over %s (type %s)", node.Assignment.Values[0], rangeExpr.String()))
				}
				keyTi := &TypeInfo{Type: key, Properties: PropertyAddressable}
				isDecl := node.Assignment.Type == ast.AssignmentDeclaration
				tc.assignSingle(node.Assignment, node.Assignment.Variables[0], nil, keyTi, nil, isDecl, false)
				if len(node.Assignment.Variables) == 2 {
					tc.assignSingle(node.Assignment, node.Assignment.Variables[1], nil, &TypeInfo{Type: elem}, nil, isDecl, false)
				}
			}
			tc.checkNodesInNewScope(node.Body)
			tc.removeLastAncestor()
			tc.removeCurrentScope()
			tc.terminating = !tc.hasBreak[node]

		case *ast.Assignment:

			tc.checkAssignment(node)
			tc.terminating = false

		case *ast.Break:

			found := false
			for i := len(tc.ancestors) - 1; i >= 0; i-- {
				switch n := tc.ancestors[i].node.(type) {
				case *ast.For, *ast.ForRange, *ast.Switch, *ast.TypeSwitch:
					tc.hasBreak[n] = true
					found = true
					break
				}
			}
			// TODO (Gianluca): remove this check from parser.
			if !found {
				panic(tc.errorf(node, "break is not in a loop, switch, or select"))
			}
			tc.terminating = false

		case *ast.Continue:
			tc.terminating = false

		case *ast.Return:

			tc.checkReturn(node)
			tc.terminating = true

		case *ast.Switch:

			terminating := true
			tc.addScope()
			tc.addToAncestors(node)
			if node.Init != nil {
				tc.checkAssignment(node.Init)
			}
			hasFallthrough := false
			hasDefault := false
			switchType := boolType
			if node.Expr != nil {
				switchType = tc.checkExpression(node.Expr).Type
			}
			for _, cas := range node.Cases {
				hasFallthrough = hasFallthrough || cas.Fallthrough
				hasDefault = hasDefault || len(cas.Expressions) == 0
				for _, expr := range cas.Expressions {
					t := tc.checkExpression(expr)
					if !isAssignableTo(t, switchType) {
						ne := ""
						if node.Expr != nil {
							ne = " on " + node.Expr.String()
						}
						panic(tc.errorf(cas, "invalid case %v in switch%s (mismatched types %s and %v)", expr, ne, t.ShortString(), switchType))
					}
				}
				tc.checkNodesInNewScope(cas.Body)
				terminating = terminating && (tc.terminating || hasFallthrough)
			}
			tc.removeLastAncestor()
			tc.removeCurrentScope()
			tc.terminating = terminating && !tc.hasBreak[node] && hasDefault

		case *ast.TypeSwitch:

			terminating := true
			tc.addScope()
			tc.addToAncestors(node)
			if node.Init != nil {
				tc.checkAssignment(node.Init)
			}
			ta := node.Assignment.Values[0].(*ast.TypeAssertion)
			t := tc.typeof(ta.Expr, noEllipses)
			if t.Type.Kind() != reflect.Interface {
				panic(tc.errorf(node, "cannot type switch on non-interface value %v (type %s)", ta.Expr, t.ShortString()))
			}
			hasDefault := false
			for _, cas := range node.Cases {
				hasDefault = hasDefault || len(cas.Expressions) == 0
				for _, expr := range cas.Expressions {
					t := tc.typeof(expr, noEllipses)
					if !t.IsType() {
						panic(tc.errorf(cas, "%v (type %s) is not a type", expr, t.StringWithNumber(true)))
					}
				}
				tc.checkNodesInNewScope(cas.Body)
				terminating = terminating && tc.terminating
			}
			tc.removeLastAncestor()
			tc.removeCurrentScope()
			tc.terminating = terminating && !tc.hasBreak[node] && hasDefault

		case *ast.Const, *ast.Var:

			tc.checkAssignment(node)
			tc.terminating = false

		case *ast.Value:

			tc.checkExpression(node.Expr)
			tc.terminating = false

		case *ast.ShowMacro:

			// TODO (Gianluca): to review.
			name := node.Macro.Name
			_, ok := tc.lookupScopes(name, false)
			if !ok {
				panic(tc.errorf("undefined macro: %s", name))
			}

		case *ast.Macro:

			// TODO (Gianluca): handle types for macros.
			name := node.Ident.Name
			_, ok := tc.lookupScopes(name, false)
			if ok {
				panic(tc.errorf("macro %s redeclared in this page", name))
			}
			tc.checkNodesInNewScope(node.Body)
			// TODO (Gianluca):
			ti := &TypeInfo{}
			tc.assignScope(name, ti)

		case *ast.Call:
			tis, isBuiltin := tc.checkCallExpression(node, true)
			if ident, ok := node.Func.(*ast.Identifier); ok {
				if isBuiltin && ident.Name == "panic" {
					tc.terminating = true
				}
				if isBuiltin && len(tis) > 0 && ident.Name != "copy" {
					panic(tc.errorf(node, "%s evaluated but not used", node))
				}
			}

		case ast.Expression:

			tc.checkExpression(node)
			panic(tc.errorf(node, "%s evaluated but not used", node))

		default:

			panic(fmt.Errorf("checkNodes not implemented for type: %T", node))

		}

	}

}

func (tc *typechecker) checkCaseExpressionSwitch(node *ast.Case, switchExpr ast.Expression) {

}

// https://golang.org/ref/spec#Return_statements
func (tc *typechecker) checkReturn(node *ast.Return) {

	fun, funcBound := tc.getCurrentFunc()
	if fun == nil {
		panic(tc.errorf(node, "non-declaration statement outside function body"))
	}

	expected := fillParametersTypes(fun.Type.Result)
	got := node.Values

	if len(expected) == 0 && len(got) == 0 {
		return
	}

	// Named return arguments with empty return: check if any value has been
	// shadowed.
	if len(expected) > 0 && expected[0].Ident != nil && len(got) == 0 {
		// If "return" belongs to an inner scope (not the function scope).
		if len(tc.scopes) > funcBound {
			for _, e := range expected {
				name := e.Ident.Name
				_, ok := tc.lookupScopes(name, true)
				if ok {
					panic(tc.errorf(node, "%s is shadowed during return", name))
				}
			}
		}
		return
	}

	expectedTypes := []reflect.Type{}
	for _, e := range expected {
		ti := tc.checkType(e.Type, noEllipses)
		expectedTypes = append(expectedTypes, ti.Type)
	}

	needsCheck := true
	if len(expected) > 1 && len(got) == 1 {
		if c, ok := got[0].(*ast.Call); ok {
			tis, _ := tc.checkCallExpression(c, false)
			got = nil
			for _, ti := range tis {
				v := ast.NewCall(c.Pos(), c.Func, c.Args, false)
				tc.typeInfo[v] = ti
				got = append(got, v)
				needsCheck = false
			}
		}
	}

	if needsCheck {
		for _, g := range got {
			_ = tc.checkExpression(g)
		}
	}

	if len(expected) != len(got) {
		msg := ""
		if len(expected) > len(got) {
			msg = "not enough arguments to return"
		}
		if len(expected) < len(got) {
			msg = "too many arguments to return"
		}
		msg += "\n\thave ("
		for i, x := range got {
			msg += tc.typeInfo[x].StringWithNumber(false)
			if i != len(got)-1 {
				msg += ", "
			}
		}
		msg += ")\n\twant ("
		for i, T := range expectedTypes {
			msg += T.String()
			if i != len(expectedTypes)-1 {
				msg += ", "
			}
		}
		msg += ")"
		panic(tc.errorf(node, msg))
	}

	for i, T := range expectedTypes {
		x := got[i]
		if !isAssignableTo(tc.typeInfo[x], T) {
			panic(tc.errorf(node, "cannot use %v (type %v) as type %v in return argument", got[i], tc.typeInfo[got[i]].ShortString(), expectedTypes[i]))
		}
	}
}
