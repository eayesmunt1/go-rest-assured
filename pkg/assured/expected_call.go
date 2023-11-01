package assured

import (
	"fmt"
)

type ExpectedCall struct {
	Path          string            `json:"path"`
	Method        string            `json:"method"`
	StatusCode    int               `json:"status_code"`
	Delay         int               `json:"delay"`
	Headers       map[string]string `json:"headers"`
	OrderedBodies *[]string         `json:"ordered_bodies"`
	Query         map[string]string `json:"query,omitempty"`
	Response      CallResponse      `json:"response,omitempty"`
	Callbacks     []Callback        `json:"callbacks,omitempty"`
}

// ID is used as a key when managing stubbed and made calls
func (c ExpectedCall) ID() string {
	return fmt.Sprintf("%s:%s", c.Method, c.Path)
}

// String converts a Call's Response into a string
func (c ExpectedCall) String() string {
	rawString := string(c.Response)

	// TODO: implement string replacements for special cases
	return rawString
}
