package espresso

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EspressoClientSuite struct {
	suite.Suite
}

func (s *EspressoClientSuite) SetupTest() {

}

func TestEspressoClientSuite(t *testing.T) {
	suite.Run(t, new(EspressoClientSuite))
}

func (s *EspressoClientSuite) TestSend() {
	// ec := EspressoClient{
	// 	EspressoUrl: "https://query.cappuccino.testnet.espresso.network",
	// 	GraphQLUrl:  "http://localhost:8080",
	// }
	// tx, err := ec.SendInputV2("aabbcc", 10008)
	// s.NoError(err)
	// fmt.Println(tx)
	// s.NotNil(tx)
}
