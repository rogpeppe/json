package main

import (
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var parseTests = []struct {
	testName    string
	args        []string
	expect      []interface{}
	expectError string
}{{
	testName: "no-args",
	args:     []string{},
	expect:   nil,
}, {
	testName: "number",
	args:     []string{"134"},
	expect:   []interface{}{134.0},
}, {
	testName: "string",
	args:     []string{"hello, world"},
	expect:   []interface{}{"hello, world"},
}, {
	testName: "null",
	args:     []string{"null"},
	expect:   []interface{}{nil},
}, {
	testName: "true",
	args:     []string{"true"},
	expect:   []interface{}{true},
}, {
	testName: "false",
	args:     []string{"false"},
	expect:   []interface{}{false},
}, {
	testName: "forced-string",
	args:     []string{"str", "1234"},
	expect:   []interface{}{"1234"},
}, {
	testName: "forced-bool",
	args:     []string{"bool", "1"},
	expect:   []interface{}{true},
}, {
	testName: "forced-number",
	args:     []string{"num", "123"},
	expect:   []interface{}{123.0},
}, {
	testName:    "forced-number-with-invalid-number",
	args:        []string{"num", "a"},
	expectError: `invalid number "a" at argument 1`,
}, {
	testName: "top-level-object",
	args:     []string{"xy:", "zw", "abc:", "de"},
	expect:   []interface{}{map[string]interface{}{"xy": "zw", "abc": "de"}},
}, {
	testName: "single-object-value",
	args:     []string{"[", "xy:", "zw", "abc:", "de", "]"},
	expect:   []interface{}{map[string]interface{}{"xy": "zw", "abc": "de"}},
}, {
	testName: "array",
	args:     []string{".[", "a", "true", "]"},
	expect:   []interface{}{[]interface{}{"a", true}},
}, {
	testName: "composite-object",
	args:     []string{"a:", "[", "b:", "4676", "c:", ".[", "1", "2", "]", "]"},
	expect: []interface{}{map[string]interface{}{
		"a": map[string]interface{}{
			"b": 4676.0,
			"c": []interface{}{1.0, 2.0},
		},
	}},
}}

func TestParse(t *testing.T) {
	c := qt.New(t)
	for _, test := range parseTests {
		c.Run(test.testName, func(c *qt.C) {
			v, err := parse(test.args)
			if test.expectError != "" {
				c.Assert(err, qt.ErrorMatches, test.expectError)
				c.Assert(v, qt.IsNil)
				return
			}
			c.Assert(err, qt.Equals, nil)
			c.Assert(v, deepEquals, test.expect)
		})
	}
}

var deepEquals = qt.CmpEquals(cmpopts.EquateApprox(1e-9, 0))
