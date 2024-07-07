package graphile

type GraphileClient interface {
	Post(requestBody []byte) ([]byte, error)
}
