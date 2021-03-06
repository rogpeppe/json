package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

var indent = flag.Bool("indent", false, "indent JSON output; by default it is printed compactly")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "json [flags] [arg...]\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `

The json command prints a sequence of JSON values specified by the command line arguments.
It's intended to make it straightforward to write JSON values on the command line.

As the brace character is special to some shells, it uses [ and ] as delimiters.
Map keys are denoted with a trailing colon.

Instead of using JSON double-quotes to delimit values, it uses shell arguments,
so all arguments including delimiters must be passed as separate arguments.

For example:

	$ json foo: 45 bar: [ x: 657 ] y: .[ 3 5 6 ]
	{"bar":{"x":657},"foo":45,"y":[3,5,6]}

The grammar is as follows (in BNF notation as used by https://golang.org/ref/spec).
All tokens represent exactly one argument on the command line. STR is any string;
KEY is a string with a ":" suffix.

	args = { value } | keyValues
	value = "null" | "true" | "false" | typeAssertion | object | array | STR
	typeAssertion = ( "str" | "num" | "bool" | "jsonstr" ) value
	object = "[" keyValues "]"
	keyValues = { key value }
	key = KEY | "key" STR
	array = ".[" { value } "]"

Note that if the top argument looks like an object key (it ends with a colon (:)),
the entire command line represents a single object; otherwise, the arguments
represent a sequence of independent objects.

Thus

	json [ a: b ]

is exactly the same as

	json a: b

A value that does not look like any of the acceptable JSON values will be treated
as a number if it looks like a number, and as a string otherwise. To ensure that
externally-provided values take on their expected type, type assertions can be used.

A type assertion asserts and/or converts its argument value
to the asserted type, and fails if the value isn't well formed for that type.
This can be used to stop arbitrary external values (for example the contents
of environment variables) from being treated as lexically significant tokens.

The possible assertions are:

	key
		The following argument is treated as a key. It must be present, but can contain
		any value. For example:

			$ json key 'a"b' hello
			{"a\"b": "hello"}
	str
		The following argument is treated as a string. It must be present, but can contain
		any value. For example:

			$ json str [
			"["
	num
		The following argument is treated as a number. The number must be well-formed.
		For example:

			$ json num bad
			json: invalid number "bad" at argument 1

	bool
		The following argument is is treated as a bool.
		It must be one of 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False.

		For example:

			$ json bool 0
			false

	jsonstr
		The following value is marshaled as JSON and used as a string value.

		For example:

			$ json jsonstr [ a: 45 b: .[ a b c } ]
			"{\"a\":45,\"b\":[\"a\",\"b\",\"c\"]}"

	json
		The following argument is treated as a JSON-encoded string
		and included as literal JSON. The string must hold well-formed JSON.
		For example:
			$  json [ one: 1 two: json '["two", 2]' ]
			{"one":1,"two":["two",2]}
`)
		os.Exit(2)
	}

	flag.Parse()
	exprs, err := parse(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "json: %s\n", err)
		os.Exit(1)
	}
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	enc := json.NewEncoder(w)
	if *indent {
		enc.SetIndent("", "\t")
	}
	for _, expr := range exprs {
		if err := enc.Encode(expr); err != nil {
			fmt.Fprintf(os.Stderr, "cannot encode value %#v: %v\n", expr, err)
			os.Exit(1)
		}
	}
}

type parser struct {
	index int
	args  []string
}

type syntaxError struct {
	e string
}

func (e *syntaxError) Error() string {
	return e.e
}

func parse(args []string) (_ []interface{}, err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if e, ok := e.(*syntaxError); ok {
			err = e
			return
		}
		panic(e)
	}()
	return parse1(&parser{args: args}), nil
}

func parse1(p *parser) []interface{} {
	a, ok := p.peek()
	if !ok {
		// No arguments -> null.
		return nil
	}
	// It's an object key; parse the whole command line as an object.
	if strings.HasSuffix(a, ":") || a == "key" {
		obj := parseKeyValues(p)
		if a, ok := p.peek(); ok {
			syntaxErrorf("unexpected argument %q at %d", a, p.index)
		}
		return []interface{}{obj}
	}
	var exprs []interface{}
	for {
		a, ok := p.peek()
		if !ok {
			return exprs
		}
		if a == "]" {
			syntaxErrorf("unexpected argument ] at %d, expected value", p.index)
		}
		exprs = append(exprs, parseValue(p))
	}
	return exprs
}

func parseKeyValues(p *parser) interface{} {
	v := make(map[string]interface{})
	for {
		key, ok := p.peek()
		if !ok || key == "]" {
			return v
		}
		if key == "key" {
			p.next()
			key = p.mustPeek("key argument")
		} else if !strings.HasSuffix(key, ":") {
			syntaxErrorf("expected object key (ending in :) or 'key' keyword at argument %d, but got %q", p.index, key)
		} else {
			key = key[0 : len(key)-1]
		}
		p.next()
		v[key] = parseValue(p)
	}
	return v
}

func parseValue(p *parser) interface{} {
	switch a := p.mustNext("value"); a {
	case "[":
		v := parseKeyValues(p)
		a := p.mustNext("]")
		if a != "]" {
			syntaxErrorf("argument %d; expected ] got %q", p.index-1, a)
		}
		return v
	case ".[":
		var v []interface{}
		for {
			if a := p.mustPeek("]"); a == "]" {
				p.next()
				break
			}
			v = append(v, parseValue(p))
		}
		return v
	case "null":
		return nil
	case "true":
		return true
	case "false":
		return false
	case "str":
		return p.mustNext("str argument")
	case "json":
		a := p.mustNext("json argument")
		dec := json.NewDecoder(strings.NewReader(a))
		dec.UseNumber()
		var x interface{}
		if err := dec.Decode(&x); err != nil {
			syntaxErrorf("cannot unmarshal json %q at argument %d", a, p.index-1)
		}
		return x
	case "jsonstr":
		v := parseValue(p)
		data, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		return string(data)
	case "num":
		a := p.mustNext("numeric value")
		n, err := strconv.ParseFloat(a, 64)
		if err != nil {
			syntaxErrorf("invalid number %q at argument %d", a, p.index-1)
		}
		if math.IsInf(n, 0) || math.IsNaN(n) {
			syntaxErrorf("%q is not a regular floating point number and cannot be encoded to JSON", a)
		}
		// Preserve the original form of the number to avoid losing precision.
		return json.Number(a)
	case "bool":
		a := p.mustNext("boolean value")
		v, err := strconv.ParseBool(a)
		if err != nil {
			syntaxErrorf("invalid boolean at argument %d: %v", p.index-1, err)
		}
		return v
	default:
		if strings.HasSuffix(a, ":") || a == "key" {
			syntaxErrorf("argument %d; expected value, got key", p.index-1)
		}
		// If it looks like a float, treat it as a float.
		n, err := strconv.ParseFloat(a, 64)
		if err == nil {
			return n
		}
		return a
	}
}

func (p *parser) mustNext(expected string) string {
	a := p.mustPeek(expected)
	p.next()
	return a
}

func (p *parser) mustPeek(expected string) string {
	a, ok := p.peek()
	if !ok {
		syntaxErrorf("unexpected end of arguments (expected %s)", expected)
	}
	return a
}

func (p *parser) next() (string, bool) {
	a, ok := p.peek()
	if !ok {
		return "", false
	}
	p.index++
	return a, true
}

func (p *parser) peek() (string, bool) {
	if p.index >= len(p.args) {
		return "", false
	}
	return p.args[p.index], true
}

func syntaxErrorf(format string, arg ...interface{}) {
	panic(&syntaxError{
		e: fmt.Sprintf(format, arg...),
	})
}
