package dataavailability

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/stretchr/testify/suite"
)

type CelestiaSuite struct {
	suite.Suite
}

func (s *CelestiaSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}

func (s *CelestiaSuite) TestSubmitBlob() {
	token := os.Getenv("CELESTIA_AUTH_TOKEN")
	url := os.Getenv("CELESTIA_URL")
	if token == "" || url == "" {
		slog.Debug("missing celestia configuration")
		return
	}
	ctx := context.Background()
	err := SubmitBlob(ctx, url, token)
	s.NoError(err)
	s.Fail("x")
}

func TestCelestiaSuite(t *testing.T) {
	suite.Run(t, &CelestiaSuite{})
}
