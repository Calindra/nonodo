package dataavailability

type HttpCustomError struct {
	status int
	body   *string
}

func NewHttpCustomError(status int, body *string) *HttpCustomError {
	return &HttpCustomError{status: status, body: body}
}

func (m *HttpCustomError) Error() string {
	return *m.body
}
func (m *HttpCustomError) Status() int {
	return m.status
}
func (m *HttpCustomError) Body() *string {
	return m.body
}
