package dataavailability

import "strconv"

type HttpCustomError struct {
	status uint
	body   *string
}

func NewHttpCustomError(status uint, body *string) *HttpCustomError {
	return &HttpCustomError{status: status, body: body}
}

func (m *HttpCustomError) Error() string {
	return "HTTP error with status " + strconv.Itoa(int(m.status)) + " and body " + *m.body
}
func (m *HttpCustomError) Status() uint {
	return m.status
}
func (m *HttpCustomError) Body() *string {
	return m.body
}
