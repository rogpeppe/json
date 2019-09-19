# json: command-line printing of JSON values

The json command prints a JSON value specified by the command line arguments.
It's intended to make it straightforward to write JSON values on the command line.
If there are no arguments, it prints `null`.

As the brace character is special to some shells, it uses `[` and `]` as delimiters.
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
