package reader

type GraphileClient interface {
	Post(requestBody []byte) ([]byte, error)
}
