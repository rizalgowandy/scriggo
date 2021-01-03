// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/internal/mapfs"
)

type html string
type css string
type js string
type json string
type markdown string

var formatTypes = map[ast.Format]reflect.Type{
	ast.FormatHTML:     reflect.TypeOf((*html)(nil)).Elem(),
	ast.FormatCSS:      reflect.TypeOf((*css)(nil)).Elem(),
	ast.FormatJS:       reflect.TypeOf((*js)(nil)).Elem(),
	ast.FormatJSON:     reflect.TypeOf((*json)(nil)).Elem(),
	ast.FormatMarkdown: reflect.TypeOf((*markdown)(nil)).Elem(),
}

var intSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(intType), Properties: propertyAddressable}
var intArrayTypeInfo = &typeInfo{Type: reflect.ArrayOf(2, intType), Properties: propertyAddressable}
var stringSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(stringType), Properties: propertyAddressable}
var stringArrayTypeInfo = &typeInfo{Type: reflect.ArrayOf(2, stringType), Properties: propertyAddressable}
var boolSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(boolType), Properties: propertyAddressable}
var boolArrayTypeInfo = &typeInfo{Type: reflect.ArrayOf(2, boolType), Properties: propertyAddressable}
var interfaceSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(emptyInterfaceType), Properties: propertyAddressable}

var stringToIntMapTypeInfo = &typeInfo{Type: reflect.MapOf(stringType, intType), Properties: propertyAddressable}
var intToStringMapTypeInfo = &typeInfo{Type: reflect.MapOf(intType, stringType), Properties: propertyAddressable}
var definedIntToStringMapTypeInfo = &typeInfo{Type: reflect.MapOf(definedIntTypeInfo.Type, stringType), Properties: propertyAddressable}

var definedIntTypeInfo = &typeInfo{Type: reflect.TypeOf(definedInt(0)), Properties: propertyAddressable}
var definedIntSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(definedIntTypeInfo.Type), Properties: propertyAddressable}

var definedStringTypeInfo = &typeInfo{Type: reflect.TypeOf(definedString("")), Properties: propertyAddressable}

var checkerTemplateExprs = []struct {
	src   string
	ti    *typeInfo
	scope map[string]*typeInfo
}{

	// contains ( slice and array )
	{`s contains 5`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo}},
	{`s contains 7.0`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo}},
	{`s contains 'c'`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo}},
	{`s contains int(2)`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo}},
	{`s contains a`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo, "a": tiInt()}},
	{`s contains b`, tiUntypedBool(), map[string]*typeInfo{"s": definedIntSliceTypeInfo, "b": definedIntTypeInfo}},
	{`s contains -2`, tiUntypedBool(), map[string]*typeInfo{"s": intArrayTypeInfo}},
	{`s contains ""`, tiUntypedBool(), map[string]*typeInfo{"s": stringSliceTypeInfo}},
	{`s contains "a"`, tiUntypedBool(), map[string]*typeInfo{"s": stringArrayTypeInfo}},
	{`s contains a`, tiUntypedBool(), map[string]*typeInfo{"s": stringArrayTypeInfo, "a": tiString()}},
	{`s contains b`, tiUntypedBool(), map[string]*typeInfo{"s": stringArrayTypeInfo, "b": tiStringConst("b")}},
	{`s contains true`, tiUntypedBool(), map[string]*typeInfo{"s": boolSliceTypeInfo}},
	{`s contains false`, tiUntypedBool(), map[string]*typeInfo{"s": boolArrayTypeInfo}},
	{`s contains bool(false)`, tiUntypedBool(), map[string]*typeInfo{"s": boolArrayTypeInfo}},
	{`s contains a`, tiUntypedBool(), map[string]*typeInfo{"s": boolArrayTypeInfo, "a": tiBool()}},
	{`s contains nil`, tiUntypedBool(), map[string]*typeInfo{"s": interfaceSliceTypeInfo}},

	// contains ( map )
	{`m contains 5`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains 7.0`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains 'c'`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains int(2)`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains a`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo, "a": tiInt()}},
	{`m contains b`, tiUntypedBool(), map[string]*typeInfo{"m": definedIntToStringMapTypeInfo, "b": definedIntTypeInfo}},
	{`m contains "a"`, tiUntypedBool(), map[string]*typeInfo{"m": stringToIntMapTypeInfo}},
	{`m contains a`, tiUntypedBool(), map[string]*typeInfo{"m": stringToIntMapTypeInfo, "a": tiString()}},

	// contains ( string and string )
	{`"ab" contains "a"`, tiUntypedBoolConst(true), nil},
	{`"ab" contains "c"`, tiUntypedBoolConst(false), nil},
	{`ab contains a`, tiUntypedBoolConst(true), map[string]*typeInfo{"ab": tiUntypedStringConst("ab"), "a": tiUntypedStringConst("a")}},
	{`ab contains b`, tiUntypedBoolConst(true), map[string]*typeInfo{"ab": tiStringConst("ab"), "b": tiStringConst("b")}},
	{`ab contains c`, tiUntypedBoolConst(false), map[string]*typeInfo{"ab": tiStringConst("ab"), "c": tiStringConst("c")}},
	{`ab contains d`, tiUntypedBoolConst(false), map[string]*typeInfo{"ab": tiUntypedStringConst("ab"), "d": tiStringConst("d")}},
	{`ab contains e`, tiUntypedBoolConst(false), map[string]*typeInfo{"ab": tiStringConst("ab"), "e": tiUntypedStringConst("e")}},
	{`ab contains f`, tiUntypedBool(), map[string]*typeInfo{"ab": tiString(), "f": tiStringConst("f")}},
	{`ab contains g`, tiUntypedBool(), map[string]*typeInfo{"ab": tiStringConst("ab"), "g": tiString()}},
	{`ab contains h`, tiUntypedBool(), map[string]*typeInfo{"ab": tiString(), "h": tiString()}},
	{`ab contains i`, tiUntypedBool(), map[string]*typeInfo{"ab": tiString(), "i": tiUntypedStringConst("i")}},
	{`ab contains j`, tiUntypedBool(), map[string]*typeInfo{"ab": tiUntypedStringConst("ab"), "j": tiString()}},
	{`ab contains k`, tiUntypedBool(), map[string]*typeInfo{"ab": definedStringTypeInfo, "k": definedStringTypeInfo}},
	{`ab contains l`, tiUntypedBool(), map[string]*typeInfo{"ab": definedStringTypeInfo, "l": tiUntypedStringConst("l")}},
	{`ab contains m`, tiUntypedBool(), map[string]*typeInfo{"ab": tiUntypedStringConst("ab"), "m": definedStringTypeInfo}},

	// contains ( string and rune )
	{`"àb" contains 'à'`, tiUntypedBoolConst(true), nil},
	{`"àb" contains 224`, tiUntypedBoolConst(true), nil},
	{`"àb" contains 'ù'`, tiUntypedBoolConst(false), nil},
	{`"àb" contains 249`, tiUntypedBoolConst(false), nil},
	{`àb contains 'à'`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiUntypedStringConst("àb")}},
	{`àb contains 'à'`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiStringConst("àb")}},
	{`àb contains 'à'`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString()}},
	{`àb contains à`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiUntypedStringConst("àb"), "à": tiUntypedRuneConst('à')}},
	{`àb contains à`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiUntypedStringConst("àb"), "à": tiRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiUntypedStringConst("àb"), "à": tiRune()}},
	{`àb contains à`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiStringConst("àb"), "à": tiUntypedRuneConst('à')}},
	{`àb contains à`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiStringConst("àb"), "à": tiRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiStringConst("àb"), "à": tiRune()}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiUntypedRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiRune()}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiUntypedIntConst("224")}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiIntConst(224)}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiInt()}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": definedStringTypeInfo, "à": tiUntypedRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": definedStringTypeInfo, "à": tiRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": definedStringTypeInfo, "à": tiRune()}},

	// macro type literal
	{`(macro() string)(nil)`, &typeInfo{Type: reflect.TypeOf((func() string)(nil))}, nil},
	{`(macro() html)(nil)`, &typeInfo{Type: reflect.TypeOf((func() html)(nil))}, nil},
	{`(macro() css)(nil)`, &typeInfo{Type: reflect.TypeOf((func() css)(nil))}, nil},
	{`(macro() js)(nil)`, &typeInfo{Type: reflect.TypeOf((func() js)(nil))}, nil},
	{`(macro() json)(nil)`, &typeInfo{Type: reflect.TypeOf((func() json)(nil))}, nil},
	{`(macro() markdown)(nil)`, &typeInfo{Type: reflect.TypeOf((func() markdown)(nil))}, nil},
}

func TestCheckerTemplateExpressions(t *testing.T) {
	options := checkerOptions{modality: templateMod, formatTypes: formatTypes}
	for _, expr := range checkerTemplateExprs {
		var lex = scanTemplate([]byte("{{ "+expr.src+" }}"), ast.FormatText)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*CheckingError); ok {
						t.Errorf("source: %q, %s\n", expr.src, err)
					} else {
						panic(r)
					}
				}
			}()
			var p = &parsing{
				lex:       lex,
				format:    ast.FormatText,
				ancestors: nil,
			}
			p.next() // discard tokenLeftBraces.
			node, tok := p.parseExpr(p.next(), false, false, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
				return
			}
			if tok.typ != tokenRightBraces {
				t.Errorf("source: %q, unexpected %s, expecting }}\n", expr.src, tok)
				return
			}
			scope := make(typeCheckerScope, len(expr.scope))
			for k, v := range expr.scope {
				scope[k] = scopeElement{t: v}
			}
			var scopes []typeCheckerScope
			if expr.scope == nil {
				scopes = []typeCheckerScope{}
			} else {
				scopes = []typeCheckerScope{scope}
			}
			tc := newTypechecker(newCompilation(), "", options, nil)
			tc.scopes = scopes
			tc.enterScope()
			ti := tc.checkExpr(node)
			err := equalTypeInfo(expr.ti, ti)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				if testing.Verbose() {
					t.Logf("\nUnexpected:\n%s\nExpected:\n%s\n", dumpTypeInfo(ti), dumpTypeInfo(expr.ti))
				}
			}
		}()
	}
}

var checkerTemplateExprErrors = []struct {
	src   string
	err   *CheckingError
	scope map[string]*typeInfo
}{

	// contains
	{`[]byte{} contains "a"`, tierr(1, 13, `invalid operation: []byte literal contains "a" (cannot convert a (type untyped string) to type uint8)`), nil},
	{`[]int{} contains int32(5)`, tierr(1, 12, `invalid operation: []int literal contains int32(5) (mismatched types int and rune)`), nil},
	{`[]int{} contains i`, tierr(1, 12, `invalid operation: []int literal contains i (mismatched types int and compiler.definedInt)`), map[string]*typeInfo{"i": definedIntTypeInfo}},
	{`[2]int{0,1} contains rune('a')`, tierr(1, 16, `invalid operation: [2]int literal contains rune('a') (mismatched types int and rune)`), nil},

	// macro type literal
	{`(macro() css)(nil)`, tierr(1, 13, `invalid macro result type css`), map[string]*typeInfo{"css": {Type: reflect.TypeOf(0), Properties: propertyIsType}}},
	{`(macro() html)(nil)`, tierr(1, 13, `invalid macro result type html`), map[string]*typeInfo{"html": {Type: reflect.TypeOf(definedInt(0)), Properties: propertyIsType}}},
	{`(macro() markdown)(nil)`, tierr(1, 13, `invalid macro result type markdown`), map[string]*typeInfo{"markdown": {Type: reflect.TypeOf(js("")), Properties: propertyIsType}}},
}

func TestCheckerTemplateExpressionErrors(t *testing.T) {
	options := checkerOptions{modality: templateMod, formatTypes: formatTypes}
	for _, expr := range checkerTemplateExprErrors {
		var lex = scanTemplate([]byte("{{ "+expr.src+" }}"), ast.FormatText)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*CheckingError); ok {
						err := sameTypeCheckError(err, expr.err)
						if err != nil {
							t.Errorf("source: %q, %s\n", expr.src, err)
							return
						}
					} else {
						panic(r)
					}
				}
			}()
			var p = &parsing{
				lex:       lex,
				format:    ast.FormatText,
				ancestors: nil,
			}
			p.next() // discard tokenLeftBraces.
			node, tok := p.parseExpr(p.next(), false, false, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
				return
			}
			if tok.typ != tokenRightBraces {
				t.Errorf("source: %q, unexpected %s, expecting }}\n", expr.src, tok)
				return
			}
			scope := make(typeCheckerScope, len(expr.scope))
			for k, v := range expr.scope {
				scope[k] = scopeElement{t: v}
			}
			var scopes []typeCheckerScope
			if expr.scope == nil {
				scopes = []typeCheckerScope{}
			} else {
				scopes = []typeCheckerScope{scope}
			}
			tc := newTypechecker(newCompilation(), "", options, nil)
			tc.scopes = scopes
			tc.enterScope()
			ti := tc.checkExpr(node)
			t.Errorf("source: %s, unexpected %s, expecting error %q\n", expr.src, ti, expr.err)
		}()
	}
}

var checkerTemplateStmts = []struct {
	src      string
	expected string
}{

	// Misc.
	{
		src:      `Just test`,
		expected: ok,
	},
	{
		src:      `{{ a }}`,
		expected: "undefined: a",
	},
	// Macro definitions.
	{
		src:      `{% macro M %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M() %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M(a, b int) %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M(int, int, string) %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M(a) %}{% end %}`,
		expected: "undefined: a",
	},
	{
		src:      `{% macro M %}{% end %}{% macro M %}{% end %}`,
		expected: "M redeclared in this block\n\tprevious declaration at 1:10",
	},

	// Show macro.
	{
		src:      `{% macro M %}{% end %}         {% show M() %}`,
		expected: ok,
	},
	{
		src:      `{% macro M %}{% end %}         {% show M(1) %}`,
		expected: "too many arguments in call to M\n\thave (number)\n\twant ()",
	},
	{
		src:      `{% macro M(int) %}{% end %}    {% show M("s") %}`,
		expected: "cannot use \"s\" (type string) as type int in argument to M",
	},

	{
		src:      `{% macro M %}{% end %}    {% show M() %}`,
		expected: ok,
	},

	{
		src:      `{% show M() %}`,
		expected: `undefined: M`,
	},

	{
		src:      `{% a := 10 %}{% a %}`,
		expected: `a evaluated but not used`,
	},

	{
		src:      `{% a := 20 %}{{ a and a }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{{ a or a }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{% b := "" %}{{ a or b and (not b) }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{{ 3 and a }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{{ 3 or a }}`,
		expected: ok,
	},

	{
		src:      `{% const a = 20 %}{{ not a }}`,
		expected: ok,
	},

	{
		src:      `{% a := true %}{% b := true %}{{ a and b or b and b }}`,
		expected: ok,
	},

	{
		src:      `{% n := 10 %}{% var a bool = not n %}`,
		expected: ok,
	},

	{
		src:      `{% a := []int(nil) %}{% if a %}{% end %}`,
		expected: ``,
	},

	{
		src:      `{% if 20 %}{% end %}`,
		expected: ``,
	},

	{
		src:      `{{ true and nil }}`,
		expected: `invalid operation: true and nil (operator 'and' not defined on nil)`,
	},

	{
		src:      `{{ nil and false }}`,
		expected: `invalid operation: nil and false (operator 'and' not defined on nil)`,
	},

	{
		src:      `{{ true or nil }}`,
		expected: `invalid operation: true or nil (operator 'or' not defined on nil)`,
	},

	{
		src:      `{{ not nil }}`,
		expected: `invalid operation: not nil (operator 'not' not defined on nil)`,
	},

	{
		src:      `{% v := 10 %}{{ v and nil }}`,
		expected: `invalid operation: v and nil (operator 'and' not defined on nil)`,
	},

	{
		// Check that the 'and' operator returns an untyped bool even if its two
		// operands are both typed booleans. The same applies to the 'or' and
		// 'not' operators.
		src: `
			{% type Bool bool %}
			{% var _ bool = Bool(true) and Bool(false) %}
		`,
		expected: ok,
	},

	{
		// Check that a format type value can be explicitly converted to
		// string.
		src: `
			{%%
				var s1 html
				var s2 css
				var s3 js
				var s4 json
				var s5 markdown
				_ = string(s1)
				_ = string(s2)
				_ = string(s3)
				_ = string(s4)
				_ = string(s5)
			%%}
		`,
		expected: ok,
	},

	{
		// Check that an untyped constant string value can be converted to a
		// format type.
		src: `
			{%%
				 const s = "a" 
				_ = html(s)
				_ = css(s)
				_ = js(s)
				_ = json(s)
				_ = markdown(s)
			%%}
		`,
		expected: ok,
	},

	{
		src:      `{% s := "a" %}{% _ = html(s) %}`,
		expected: `cannot convert s (type string) to type compiler.html`,
	},

	{
		src:      `{% s := "a" %}{% _ = css(s) %}`,
		expected: `cannot convert s (type string) to type compiler.css`,
	},

	{
		src:      `{% s := "a" %}{% _ = js(s) %}`,
		expected: `cannot convert s (type string) to type compiler.js`,
	},

	{
		src:      `{% s := "a" %}{% _ = json(s) %}`,
		expected: `cannot convert s (type string) to type compiler.json`,
	},

	{
		src:      `{% s := "a" %}{% _ = markdown(s) %}`,
		expected: `cannot convert s (type string) to type compiler.markdown`,
	},

	{
		// Check that an typed format constant can be converted to the same
		// format type.
		src: `
			{%%
				const s1 html = "a"
				const s2 css = "a"
				const s3 js = "a"
				const s4 json = "a"
				const s5 markdown = "a"
				_ = html(s1)
				_ = css(s2)
				_ = js(s3)
				_ = json(s4)
				_ = markdown(s5)
			%%}
		`,
		expected: ok,
	},

	{
		// Check that a non-constant format value can be converted to the same
		// format type.
		src: `
			{%%
				var s1 html
				var s2 css
				var s3 js
				var s4 json
				var s5 markdown
				_ = html(s1)
				_ = css(s2)
				_ = js(s3)
				_ = json(s4)
				_ = markdown(s5)
			%%}
		`,
		expected: ok,
	},
}

func TestCheckerTemplatesStatements(t *testing.T) {
	options := Options{FormatTypes: formatTypes}
	for _, cas := range checkerTemplateStmts {
		src := cas.src
		expected := cas.expected
		t.Run(src, func(t *testing.T) {
			fsys := mapfs.MapFS{"index.html": src}
			_, err := BuildTemplate(fsys, "index.html", options)
			switch {
			case expected == "" && err != nil:
				t.Fatalf("unexpected error: %q", err)
			case expected != "" && err == nil:
				t.Fatalf("expecting error %q, got nothing", expected)
			case expected != "" && err != nil && !strings.Contains(err.Error(), expected):
				t.Fatalf("expecting error %q, got %q", expected, err.Error())
			}
		})
	}
}
