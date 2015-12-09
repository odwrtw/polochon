package errors

import "fmt"

type Multiple struct {
	Errors []*Error
	fatal  *Error
}

func NewMultiple() *Multiple {
	return &Multiple{}
}

func (me *Multiple) Error() string {
	if me.fatal != nil {
		return me.fatal.Error()
	}
	if len(me.Errors) == 1 {
		return me.Errors[0].Error()
	}
	return fmt.Sprintf("Got %d errors", len(me.Errors))
}

func (me *Multiple) Fatal(err error) {
	me.fatal = Wrap(err, 1)
}

func (me *Multiple) IsFatal() bool {
	return me.fatal != nil
}

func (me *Multiple) Add(i interface{}) {
	err := Wrap(i, 1)
	me.Errors = append(me.Errors, err)
}

func (me *Multiple) AddWithContext(i interface{}, ctx Context) {
	err := Wrap(i, 1)
	err.AddContext(ctx)
	me.Errors = append(me.Errors, err)
}

func (me *Multiple) HasError() bool {
	return len(me.Errors) != 0
}
