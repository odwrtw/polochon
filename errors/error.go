package errors

import (
	"bytes"
	"fmt"
	"runtime"

	"github.com/Sirupsen/logrus"
)

// The maximum number of stackframes on any error.
var MaxStackDepth = 50

// Context type, used to pass to `WithContext`.
type Context map[string]interface{}

// Error implements error interface
type Error struct {
	msg    string
	stack  []uintptr
	fatal  bool
	ctx    Context
	frames []StackFrame
}

// New returns a new Error pointer
func New(format string, a ...interface{}) *Error {
	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2, stack[:])

	return &Error{
		msg:   fmt.Sprintf(format, a...),
		stack: stack[:length],
		fatal: false,
		ctx:   Context{},
	}
}

// AddContext adds context to the Error
func (e *Error) AddContext(ctx Context) *Error {
	for k, v := range ctx {
		e.ctx[k] = v
	}
	return e
}

// Ctx is a shortcut for AddContext for adding a single
// key/val pair
func (e *Error) Ctx(key string, val interface{}) *Error {
	e.ctx[key] = val
	return e
}

// GetCtx returns error context
func (e *Error) GetCtx() Context {
	return e.ctx
}

// Error implements error golang interface
func (e *Error) Error() string {
	return e.msg
}

// Stack returns the callstack formatted the same way that go does
// in runtime/debug.Stack()
func (e *Error) Stack() []byte {
	buf := bytes.Buffer{}

	for _, frame := range e.StackFrames() {
		buf.WriteString(frame.String())
	}

	return buf.Bytes()
}

// ErrorStack returns a string that contains both the
// error message and the callstack.
func (e *Error) ErrorStack() string {
	return e.Error() + "\n" + string(e.Stack())
}

// StackFrames returns an array of frames containing information about the
// stack.
func (e *Error) StackFrames() []StackFrame {
	if e.frames == nil {
		e.frames = make([]StackFrame, len(e.stack))

		for i, pc := range e.stack {
			e.frames[i] = NewStackFrame(pc)
		}
	}

	return e.frames
}

// Fatal set error as fatal
func (e *Error) Fatal() *Error {
	e.fatal = true
	return e
}

// IsFatal returns true if the error is a fatal one
func (e *Error) IsFatal() bool {
	return e.fatal
}

// Wrap is a shortcut for WrapSkip(e, 0)
func Wrap(e interface{}) *Error {
	return WrapSkip(e, 1)
}

// WrapSkip takes an error or a string and return an Error
// object
func WrapSkip(e interface{}, skip int) *Error {
	var err error
	switch e := e.(type) {
	case *Error:
		return e
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}
	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+skip, stack[:])

	return &Error{
		msg:   err.Error(),
		stack: stack[:length],
		fatal: false,
		ctx:   Context{},
	}

}

// LogErrors logs error with debug information
func LogErrors(log *logrus.Entry, e error) {
	switch err := e.(type) {
	case *Error:
		if err.IsFatal() {
			log.WithFields(logrus.Fields(err.GetCtx())).Error(err.Error())
			log.WithFields(logrus.Fields(err.GetCtx())).Debug(err.ErrorStack())
		} else {
			log.WithFields(logrus.Fields(err.GetCtx())).Warning(err.Error())
			log.WithFields(logrus.Fields(err.GetCtx())).Debug(err.ErrorStack())
		}
	case *Collector:
		if err.IsFatal() {
			for _, e := range err.Errors {
				log.WithFields(logrus.Fields(e.GetCtx())).Error(e.Error())
				log.WithFields(logrus.Fields(e.GetCtx())).Debug(e.ErrorStack())
			}
		} else {
			for _, e := range err.Errors {
				log.WithFields(logrus.Fields(e.GetCtx())).Warning(e.Error())
				log.WithFields(logrus.Fields(e.GetCtx())).Debug(e.ErrorStack())
			}

		}
	case error:
		log.Error(err.Error())
	}

}

// IsFatal checks if the error is fatal
func IsFatal(e error) bool {
	switch err := e.(type) {
	case *Error:
		return err.IsFatal()
	case *Collector:
		return err.IsFatal()
	default:
		return true
	}
}
