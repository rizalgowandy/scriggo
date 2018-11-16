//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"testing"

	"open2b/template/ast"
)

var typeTests = map[string][]tokenType{
	``:                                                 {},
	`a`:                                                {tokenText},
	`{`:                                                {tokenText},
	`}`:                                                {tokenText},
	`{{a}}`:                                            {tokenStartValue, tokenIdentifier, tokenEndValue},
	`{{ a }}`:                                          {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{\ta\n}}":                                        {tokenStartValue, tokenIdentifier, tokenSemicolon, tokenEndValue},
	"{{\na\t}}":                                        {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{\na;\t}}":                                       {tokenStartValue, tokenIdentifier, tokenSemicolon, tokenEndValue},
	"{% var a = 1 %}":                                  {tokenStartStatement, tokenVar, tokenIdentifier, tokenAssignment, tokenNumber, tokenEndStatement},
	"{% a = 2 %}":                                      {tokenStartStatement, tokenIdentifier, tokenAssignment, tokenNumber, tokenEndStatement},
	"{%for()%}":                                        {tokenStartStatement, tokenFor, tokenLeftParenthesis, tokenRightParenthesis, tokenEndStatement},
	"{%\tfor()\n%}":                                    {tokenStartStatement, tokenFor, tokenLeftParenthesis, tokenRightParenthesis, tokenSemicolon, tokenEndStatement},
	"{%\tfor a%}":                                      {tokenStartStatement, tokenFor, tokenIdentifier, tokenEndStatement},
	"{% for a;\n\t%}":                                  {tokenStartStatement, tokenFor, tokenIdentifier, tokenSemicolon, tokenEndStatement},
	"{%end%}":                                          {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{%\tend\n%}":                                      {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% end %}":                                        {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% break %}":                                      {tokenStartStatement, tokenBreak, tokenEndStatement},
	"{% continue %}":                                   {tokenStartStatement, tokenContinue, tokenEndStatement},
	"{% if a %}":                                       {tokenStartStatement, tokenIf, tokenIdentifier, tokenEndStatement},
	"{% else %}":                                       {tokenStartStatement, tokenElse, tokenEndStatement},
	"{% extend \"\" %}":                                {tokenStartStatement, tokenExtend, tokenInterpretedString, tokenEndStatement},
	"{% region \"\" %}":                                {tokenStartStatement, tokenRegion, tokenInterpretedString, tokenEndStatement},
	"{% show \"\" %}":                                  {tokenStartStatement, tokenShow, tokenInterpretedString, tokenEndStatement},
	"{# comment #}":                                    {tokenComment},
	`a{{b}}c`:                                          {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	`{{a}}c`:                                           {tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	`{{a}}\n`:                                          {tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	`{{a}}{{b}}`:                                       {tokenStartValue, tokenIdentifier, tokenEndValue, tokenStartValue, tokenIdentifier, tokenEndValue},
	"<script></script>":                                {tokenText},
	"<style></style>":                                  {tokenText},
	"<script>{{a}}</script>":                           {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	"<style>{{a}}</style>":                             {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	"<a class=\"{{c}}\"></a>":                          {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	"{{ _ }}":                                          {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ __ }}":                                         {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ _a }}":                                         {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ a5 }}":                                         {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ 3 }}":                                          {tokenStartValue, tokenNumber, tokenEndValue},
	"{{ -3 }}":                                         {tokenStartValue, tokenSubtraction, tokenNumber, tokenEndValue},
	"{{ +3 }}":                                         {tokenStartValue, tokenAddition, tokenNumber, tokenEndValue},
	"{{ 0 }}":                                          {tokenStartValue, tokenNumber, tokenEndValue},
	"{{ 2147483647 }}":                                 {tokenStartValue, tokenNumber, tokenEndValue},
	"{{ -2147483648 }}":                                {tokenStartValue, tokenSubtraction, tokenNumber, tokenEndValue},
	"{{ .0 }}":                                         {tokenStartValue, tokenNumber, tokenEndValue},
	"{{ 0. }}":                                         {tokenStartValue, tokenNumber, tokenEndValue},
	"{{ 0.0 }}":                                        {tokenStartValue, tokenNumber, tokenEndValue},
	"{{ 2147483647.2147483647 }}":                      {tokenStartValue, tokenNumber, tokenEndValue},
	"{{ -2147483647.2147483647 }}":                     {tokenStartValue, tokenSubtraction, tokenNumber, tokenEndValue},
	"{{ 2147483647.2147483647214748364721474836472 }}": {tokenStartValue, tokenNumber, tokenEndValue},
	"{{ a + b }}":                                      {tokenStartValue, tokenIdentifier, tokenAddition, tokenIdentifier, tokenEndValue},
	"{{ a - b }}":                                      {tokenStartValue, tokenIdentifier, tokenSubtraction, tokenIdentifier, tokenEndValue},
	"{{ a * b }}":                                      {tokenStartValue, tokenIdentifier, tokenMultiplication, tokenIdentifier, tokenEndValue},
	"{{ a / b }}":                                      {tokenStartValue, tokenIdentifier, tokenDivision, tokenIdentifier, tokenEndValue},
	"{{ a % b }}":                                      {tokenStartValue, tokenIdentifier, tokenModulo, tokenIdentifier, tokenEndValue},
	"{{ ( a ) }}":                                      {tokenStartValue, tokenLeftParenthesis, tokenIdentifier, tokenRightParenthesis, tokenEndValue},
	"{{ a == b }}":                                     {tokenStartValue, tokenIdentifier, tokenEqual, tokenIdentifier, tokenEndValue},
	"{{ a != b }}":                                     {tokenStartValue, tokenIdentifier, tokenNotEqual, tokenIdentifier, tokenEndValue},
	"{{ a && b }}":                                     {tokenStartValue, tokenIdentifier, tokenAnd, tokenIdentifier, tokenEndValue},
	"{{ a || b }}":                                     {tokenStartValue, tokenIdentifier, tokenOr, tokenIdentifier, tokenEndValue},
	"{{ a < b }}":                                      {tokenStartValue, tokenIdentifier, tokenLess, tokenIdentifier, tokenEndValue},
	"{{ a <= b }}":                                     {tokenStartValue, tokenIdentifier, tokenLessOrEqual, tokenIdentifier, tokenEndValue},
	"{{ a > b }}":                                      {tokenStartValue, tokenIdentifier, tokenGreater, tokenIdentifier, tokenEndValue},
	"{{ a >= b }}":                                     {tokenStartValue, tokenIdentifier, tokenGreaterOrEqual, tokenIdentifier, tokenEndValue},
	"{{ !a }}":                                         {tokenStartValue, tokenNot, tokenIdentifier, tokenEndValue},
	"{{ a[5] }}":                                       {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenNumber, tokenRightBrackets, tokenEndValue},
	"{{ a[:] }}":                                       {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenRightBrackets, tokenEndValue},
	"{{ a[:8] }}":                                      {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenNumber, tokenRightBrackets, tokenEndValue},
	"{{ a[3:] }}":                                      {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenNumber, tokenColon, tokenRightBrackets, tokenEndValue},
	"{{ a[3:8] }}":                                     {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenNumber, tokenColon, tokenNumber, tokenRightBrackets, tokenEndValue},
	"{{ a() }}":                                        {tokenStartValue, tokenIdentifier, tokenLeftParenthesis, tokenRightParenthesis, tokenEndValue},
	"{{ a(1) }}":                                       {tokenStartValue, tokenIdentifier, tokenLeftParenthesis, tokenNumber, tokenRightParenthesis, tokenEndValue},
	"{{ a(1,2) }}":                                     {tokenStartValue, tokenIdentifier, tokenLeftParenthesis, tokenNumber, tokenComma, tokenNumber, tokenRightParenthesis, tokenEndValue},
	"{{ a.b }}":                                        {tokenStartValue, tokenIdentifier, tokenPeriod, tokenIdentifier, tokenEndValue},
	"{{ \"\" }}":                                       {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\u09AF\" }}":                                {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\u09af\" }}":                                {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\U00008a9e\" }}":                            {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\U0010FFFF\" }}":                            {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\t\" }}":                                    {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\u3C2E\\\"\" }}":                            {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ `` }}":                                         {tokenStartValue, tokenRawString, tokenEndValue},
	"{{ `\\t` }}":                                      {tokenStartValue, tokenRawString, tokenEndValue},
	"{{ ( 1 + 2 ) * 3 }}": {tokenStartValue, tokenLeftParenthesis, tokenNumber, tokenAddition, tokenNumber, tokenRightParenthesis,
		tokenMultiplication, tokenNumber, tokenEndValue},
}

var contextTests = map[ast.Context]map[string][]ast.Context{
	ast.ContextText: {
		`a`:                             {ast.ContextText},
		`{{a}}`:                         {ast.ContextText, ast.ContextText, ast.ContextText},
		"<script></script>":             {ast.ContextText},
		"<style></style>":               {ast.ContextText},
		"<script>s{{a}}t</script>{{a}}": {ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText},
		"<style>s{{a}}t</style>{{a}}":   {ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText},
	},
	ast.ContextHTML: {
		`a`:                                        {ast.ContextText},
		`{{a}}`:                                    {ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		"<script></script>":                        {ast.ContextText},
		"<style></style>":                          {ast.ContextText},
		"<script>s{{a}}</script>{{a}}":             {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		"<style>s{{a}}</style>{{a}}":               {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style>s{{show "a"}}t</style>`:            {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<script>s{{show "a"}}t</script>`:          {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText},
		`<style a="b">{{1}}</style>`:               {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<script a="b">{{1}}</script>`:             {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText},
		`<![CDATA[<script>{{1}}</script>]]>`:       {ast.ContextText},
		`a<![CDATA[<script>{{1}}</script>]]>{{2}}`: {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
	},
	ast.ContextCSS: {
		`a`:                             {ast.ContextText},
		`{{a}}`:                         {ast.ContextCSS, ast.ContextCSS, ast.ContextCSS},
		"<script></script>":             {ast.ContextText},
		"<style></style>":               {ast.ContextText},
		"<script>s{{a}}t</script>{{a}}": {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS},
		"<style>s{{a}}t</style>{{a}}":   {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS},
	},
	ast.ContextJavaScript: {
		`a`:                             {ast.ContextText},
		`{{a}}`:                         {ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript},
		"<script></script>":             {ast.ContextText},
		"<style></style>":               {ast.ContextText},
		"<script>s{{a}}t</script>{{a}}": {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript},
		"<style>s{{a}}t</style>{{a}}":   {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript},
	},
}

var positionTests = []struct {
	src string
	pos []ast.Position
}{
	{"a", []ast.Position{
		{1, 1, 0, 0}}},
	{"\n", []ast.Position{
		{1, 1, 0, 0}}},
	{"{{a}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 4}}},
	{"\n{{a}}", []ast.Position{
		{1, 1, 0, 0},
		{2, 1, 1, 2}, {2, 3, 3, 3}, {2, 4, 4, 5}}},
	{"{{a.b}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 3}, {1, 5, 4, 4}, {1, 6, 5, 6}}},
	{"{{1\t+\n2}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 5, 4, 4}, {2, 1, 6, 6}, {2, 2, 7, 8}}},
	{"{{a}}\n{{b}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 4}, {1, 6, 5, 5},
		{2, 1, 6, 7}, {2, 3, 8, 8}, {2, 4, 9, 10}}},
	{"{{a}}\n<b\nc=\"{{a}}\">\n{{a}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 4}, {1, 6, 5, 11},
		{3, 4, 12, 13}, {3, 6, 14, 14}, {3, 7, 15, 16}, {3, 9, 17, 19},
		{4, 1, 20, 21}, {4, 3, 22, 22}, {4, 4, 23, 24}}},
	{"a<![CDATA[a\nb]]>b{{a}}", []ast.Position{
		{1, 1, 0, 16}, {2, 6, 17, 18}, {2, 8, 19, 19}, {2, 9, 20, 21}}},
	{"a{# a #}b", []ast.Position{
		{1, 1, 0, 0}, {1, 2, 1, 7}, {1, 9, 8, 8}}},
	{"a{# 本 #}b", []ast.Position{
		{1, 1, 0, 0}, {1, 2, 1, 9}, {1, 9, 10, 10}}},
}

func TestLexerTypes(t *testing.T) {
TYPES:
	for source, types := range typeTests {
		var lex = newLexer([]byte(source), ast.ContextText)
		var i int
		for tok := range lex.tokens {
			if tok.typ == tokenEOF {
				break
			}
			if i >= len(types) {
				t.Errorf("source: %q, unexpected %s\n", source, tok)
				continue TYPES
			}
			if tok.typ != types[i] {
				t.Errorf("source: %q, unexpected %s, expecting %s\n", source, tok, types[i])
				continue TYPES
			}
			i++
		}
		if lex.err != nil {
			t.Errorf("source: %q, error %s\n", source, lex.err)
		}
		if i < len(types) {
			t.Errorf("source: %q, less types\n", source)
		}
	}
}

func TestLexerContexts(t *testing.T) {
CONTEXTS:
	for ctx, tests := range contextTests {
		for source, contexts := range tests {
			var lex = newLexer([]byte(source), ctx)
			var i int
			for tok := range lex.tokens {
				if tok.typ == tokenEOF {
					break
				}
				if i >= len(contexts) {
					t.Errorf("source: %q, missing context in test\n", source)
					continue CONTEXTS
				}
				if tok.ctx != contexts[i] {
					t.Errorf("source: %q, for %s unexpected %s, expecting %s\n", source, tok, tok.ctx, contexts[i])
					continue CONTEXTS
				}
				i++
			}
			if lex.err != nil {
				t.Errorf("source: %q, error %s\n", source, lex.err)
			}
			if i < len(contexts) {
				t.Errorf("source: %q, less contexts\n", source)
			}
		}
	}
}

func TestPositions(t *testing.T) {
	for _, test := range positionTests {
		var lex = newLexer([]byte(test.src), ast.ContextHTML)
		var i int
		for tok := range lex.tokens {
			if tok.typ == tokenEOF {
				break
			}
			pos := test.pos[i]
			if tok.pos.Line != pos.Line {
				t.Errorf("source: %q, token: %s, unexpected line %d, expecting %d\n",
					test.src, tok.String(), tok.pos.Line, pos.Line)
			}
			if tok.pos.Column != pos.Column {
				t.Errorf("source: %q, token: %s, unexpected column %d, expecting %d\n",
					test.src, tok.String(), tok.pos.Column, pos.Column)
			}
			if tok.pos.Start != pos.Start {
				t.Errorf("source: %q, token: %s, unexpected start %d, expecting %d\n",
					test.src, tok.String(), tok.pos.Start, pos.Start)
			}
			if tok.pos.End != pos.End {
				t.Errorf("source: %q, token: %s, unexpected end %d, expecting %d\n",
					test.src, tok.String(), tok.pos.End, pos.End)
			}
			i++
		}
		if lex.err != nil {
			t.Errorf("source: %q, error %s\n", test.src, lex.err)
		}
		if i < len(test.pos) {
			t.Errorf("source: %q, less lines\n", test.src)
		}
	}
}
