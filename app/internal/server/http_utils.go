package server

import "net/http"

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
func (s *Server) renderError(w http.ResponseWriter, e error) {
	var err *Error

	custErr, ok := e.(*Error)
	if !ok {
		err = &Error{
			Code:    http.StatusInternalServerError,
			Message: e.Error(),
		}
	} else {
		err = custErr
	}

	s.render.JSON(w, err.Code, err)
}
