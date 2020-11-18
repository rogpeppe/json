# json: command-line printing of JSON values

The json command prints a sequence of JSON values specified by the command line arguments.
It's intended to make it straightforward to write JSON values on the command line.

As the brace character is special to some shells, it uses `[` and `]` as delimiters.
Map keys are denoted with a trailing colon.

Instead of using JSON double-quotes to delimit values, it uses shell arguments,
so all arguments including delimiters must be passed as separate arguments.

For example:

	$ json foo: 45 bar: [ x: 657 ] y: .[ 3 5 6 ]
	{"bar":{"x":657},"foo":45,"y":[3,5,6]}

The grammar is as follows (in BNF notation as used by https://golang.org/ref/spec).
All tokens represent exactly one argument on the command line. STR is any argument;
KEY is an argument with a ":" suffix.

	args = { value } | keyValues
	value = "null" | "true" | "false" | typeAssertion | object | array | STR
	typeAssertion = ( "str" | "num" | "bool" | "jsonstr" ) value
	object = "[" keyValues "]"
	keyValues = { key value }
	key = KEY | "key" STR
	array = ".[" { value } "]"

Note that if the first argument looks like an object key (it ends with a colon (:) or
is the literal string "key"),
the entire command line represents a single object; otherwise, the arguments
represent a sequence of independent objects.

Thus

	json [ a: b ]

is exactly the same as

	json a: b

A value that does not look like any of the acceptable JSON values or an object key will be treated
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

	json
		The following argument is treated as a JSON-encoded string
		and included as literal JSON. The string must hold well-formed JSON.
