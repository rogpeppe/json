package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
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

The json command prints a JSON value specified by the command line arguments.
It's intended to make it straightforward to write JSON values on the command line.
If there are no arguments, it prints null.

As the brace character is special to some shells, it uses [ and ] as delimiters.
Map keys are denoted with a trailing colon.

Instead of using JSON double-quotes to delimit values, it uses shell arguments,
so all arguments including delimiters must be passed as separate arguments.

For example:

	$ json foo: 45 bar: [ x: 657 ] y: .[ 3 5 6 ]
	{"bar":{"x":657},"foo":45,"y":[3,5,6]}

The grammar is as follows (in BNF notation as used by https://golang.org/ref/spec).
All tokens represent exactly one argument on the command line:

	args = keyValues | value
	keyValues = { KEY value }
	value = "null" | "true" | "false" | typeAssertion | object | array | STR
	typeAssertion = ( "str" | "num" | "bool" | "jsonstr" ) value
	object = "[" keyValues "]"
	array = ".[" { value } "]"

A value that does not look like any of the acceptable JSON values will be treated
as a number if it looks like a number, and as a string otherwise. To ensure that
externally-provided values take on their expected type, type assertions can be used.

A type assertion asserts and/or converts its argument value
to the asserted type, and fails if the value isn't well formed for that type.
This can be used to stop arbitrary external values (for example the contents
of environment variables) from being treated as lexically significant tokens.

The possible assertions are:

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
`)
		os.Exit(2)
	}

	flag.Parse()
	expr, err := parse(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "json: %s\n", err)
		os.Exit(1)
	}
	var data []byte
	if *indent {
		data, err = json.MarshalIndent(expr, "", "\t")
	} else {
		data, err = json.Marshal(expr)
	}
	if err != nil {
		// Should never happen.
		log.Fatal("json: ", err)
	}
	fmt.Printf("%s\n", data)
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

func parse(args []string) (_ interface{}, err error) {
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

func parse1(p *parser) interface{} {
	a, ok := p.peek()
	if !ok {
		// No arguments -> null.
		return nil
	}
	// It's an object key; parse the whole command line as an object.
	if strings.HasSuffix(a, ":") {
		obj := parseKeyValues(p)
		if a, ok := p.peek(); ok {
			syntaxErrorf("unexpected argument %q at %d", a, p.index)
		}
		return obj
	}
	obj := parseValue(p)
	if a, ok := p.peek(); ok {
		syntaxErrorf("unexpected argument %q at %d (multiple top level arguments must form an object)", a, p.index)
	}
	return obj
}

func parseKeyValues(p *parser) interface{} {
	v := make(map[string]interface{})
	for {
		key, ok := p.peek()
		if !ok || key == "]" {
			return v
		}
		if !strings.HasSuffix(key, ":") {
			syntaxErrorf("expected object key (ending in :) at argument %d, but got %q", p.index, key)
		}
		p.next()
		key = key[0 : len(key)-1]
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
		return n
	case "bool":
		a := p.mustNext("boolean value")
		v, err := strconv.ParseBool(a)
		if err != nil {
			syntaxErrorf("invalid boolean at argument %d: %v", p.index-1, err)
		}
		return v
	default:
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
