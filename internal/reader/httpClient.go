package reader

type HttpClient interface {
	Post(requestBody []byte) ([]byte, error)
}
