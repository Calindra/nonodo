package convenience

import "github.com/ethereum/go-ethereum/common"

// Voucher metadata type
type ConvenienceVoucher struct {
	Destination common.Address `db:"Destination"`
	Payload     string         `db:"Payload"`
	// type based on https://github.com/cartesi/rollups-node/blob/392c75972037352ecf94fb482619781b1b09083f/offchain/rollups-events/src/rollups_outputs.rs#L41
	InputIndex  uint64 `db:"InputIndex"`
	OutputIndex uint64 `db:"OutputIndex"`

	Executed bool `db:"Executed"`

	// Proof we can fetch from the original GraphQL

	// future improvements
	// Contract        common.Address
	// Beneficiary     common.Address
	// Label           string
	// Amount          uint64
	// ExecutedAt      uint64
	// ExecutedBlock   uint64
	// InputIndex      int
	// OutputIndex     int
	// MethodSignature string
	// ERCX            string
}
