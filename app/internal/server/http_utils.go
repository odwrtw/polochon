package server

import (
	"net/http"

	"github.com/odwrtw/polochon/lib/media_index"
)

// Error represents an http error
type Error struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
}

// Error implements the error interface
func (err *Error) Error() string {
	return err.Message
}

func (s *Server) renderOK(w http.ResponseWriter, i interface{}) {
	s.render.JSON(w, http.StatusOK, i)
}

// renderError renders the errors as JSON
func (s *Server) renderError(w http.ResponseWriter, input error) {
	var err *Error

	s.log.Error(input)

	switch e := input.(type) {
	case *Error:
		err = e
	default:
		if e == index.ErrNotFound {
			err = &Error{
				Code:    http.StatusNotFound,
				Message: "URL not found",
			}
		} else {
			err = &Error{
				Code:    http.StatusInternalServerError,
				Message: "internal error",
			}
		}
	}

	s.render.JSON(w, err.Code, err)
}
