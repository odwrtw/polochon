package errors

import "fmt"

// Collector handle multiple errors
type Collector struct {
	Errors []*Error
}

// NewCollector returns a new Collector
func NewCollector() *Collector {
	return &Collector{
		Errors: []*Error{},
	}
}

// Push adds Error to the Collector
func (c *Collector) Push(e *Error) {
	c.Errors = append(c.Errors, e)
}

// IsFatal returns true if one error is a fatal one
func (c *Collector) IsFatal() bool {
	for _, e := range c.Errors {
		if e.IsFatal() {
			return true
		}
	}
	return false
}

// HasErrors returns true if there are errors in the
// collector
func (c *Collector) HasErrors() bool {
	if len(c.Errors) == 0 {
		return false
	}
	return true
}

// Error implements error golang interface
func (c *Collector) Error() string {
	switch len(c.Errors) {
	case 0:
		return fmt.Sprintf("No error")
	case 1:
		return c.Errors[0].Error()
	}

	str := fmt.Sprintf("Got %d errors:\n", len(c.Errors))
	for _, e := range c.Errors {
		str = fmt.Sprintf("%s%s\n", str, e.Error())
	}
	return str
}
