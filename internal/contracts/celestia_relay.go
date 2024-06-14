// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// AttestationProof is an auto generated low-level Go binding around an user-defined struct.
type AttestationProof struct {
	TupleRootNonce *big.Int
	Tuple          DataRootTuple
	Proof          BinaryMerkleProof
}

// BinaryMerkleProof is an auto generated low-level Go binding around an user-defined struct.
type BinaryMerkleProof struct {
	SideNodes [][32]byte
	Key       *big.Int
	NumLeaves *big.Int
}

// DataRootTuple is an auto generated low-level Go binding around an user-defined struct.
type DataRootTuple struct {
	Height   *big.Int
	DataRoot [32]byte
}

// Namespace is an auto generated low-level Go binding around an user-defined struct.
type Namespace struct {
	Version [1]byte
	Id      [28]byte
}

// NamespaceMerkleMultiproof is an auto generated low-level Go binding around an user-defined struct.
type NamespaceMerkleMultiproof struct {
	BeginKey  *big.Int
	EndKey    *big.Int
	SideNodes []NamespaceNode
}

// NamespaceNode is an auto generated low-level Go binding around an user-defined struct.
type NamespaceNode struct {
	Min    Namespace
	Max    Namespace
	Digest [32]byte
}

// SharesProof is an auto generated low-level Go binding around an user-defined struct.
type SharesProof struct {
	Data             [][]byte
	ShareProofs      []NamespaceMerkleMultiproof
	Namespace        Namespace
	RowRoots         []NamespaceNode
	RowProofs        []BinaryMerkleProof
	AttestationProof AttestationProof
}

// CelestiaRelayMetaData contains all meta data concerning the CelestiaRelay contract.
var CelestiaRelayMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIInputBox\",\"name\":\"_inputBox\",\"type\":\"address\"},{\"internalType\":\"contractIDAOracle\",\"name\":\"_blobstreamX\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_dapp\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"bytes[]\",\"name\":\"data\",\"type\":\"bytes[]\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"beginKey\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"endKey\",\"type\":\"uint256\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes1\",\"name\":\"version\",\"type\":\"bytes1\"},{\"internalType\":\"bytes28\",\"name\":\"id\",\"type\":\"bytes28\"}],\"internalType\":\"structNamespace\",\"name\":\"min\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes1\",\"name\":\"version\",\"type\":\"bytes1\"},{\"internalType\":\"bytes28\",\"name\":\"id\",\"type\":\"bytes28\"}],\"internalType\":\"structNamespace\",\"name\":\"max\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"digest\",\"type\":\"bytes32\"}],\"internalType\":\"structNamespaceNode[]\",\"name\":\"sideNodes\",\"type\":\"tuple[]\"}],\"internalType\":\"structNamespaceMerkleMultiproof[]\",\"name\":\"shareProofs\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"bytes1\",\"name\":\"version\",\"type\":\"bytes1\"},{\"internalType\":\"bytes28\",\"name\":\"id\",\"type\":\"bytes28\"}],\"internalType\":\"structNamespace\",\"name\":\"namespace\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes1\",\"name\":\"version\",\"type\":\"bytes1\"},{\"internalType\":\"bytes28\",\"name\":\"id\",\"type\":\"bytes28\"}],\"internalType\":\"structNamespace\",\"name\":\"min\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes1\",\"name\":\"version\",\"type\":\"bytes1\"},{\"internalType\":\"bytes28\",\"name\":\"id\",\"type\":\"bytes28\"}],\"internalType\":\"structNamespace\",\"name\":\"max\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"digest\",\"type\":\"bytes32\"}],\"internalType\":\"structNamespaceNode[]\",\"name\":\"rowRoots\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"bytes32[]\",\"name\":\"sideNodes\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"key\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"numLeaves\",\"type\":\"uint256\"}],\"internalType\":\"structBinaryMerkleProof[]\",\"name\":\"rowProofs\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tupleRootNonce\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"dataRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structDataRootTuple\",\"name\":\"tuple\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32[]\",\"name\":\"sideNodes\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"key\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"numLeaves\",\"type\":\"uint256\"}],\"internalType\":\"structBinaryMerkleProof\",\"name\":\"proof\",\"type\":\"tuple\"}],\"internalType\":\"structAttestationProof\",\"name\":\"attestationProof\",\"type\":\"tuple\"}],\"internalType\":\"structSharesProof\",\"name\":\"_proof\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"_root\",\"type\":\"bytes32\"}],\"name\":\"relayShares\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// CelestiaRelayABI is the input ABI used to generate the binding from.
// Deprecated: Use CelestiaRelayMetaData.ABI instead.
var CelestiaRelayABI = CelestiaRelayMetaData.ABI

// CelestiaRelay is an auto generated Go binding around an Ethereum contract.
type CelestiaRelay struct {
	CelestiaRelayCaller     // Read-only binding to the contract
	CelestiaRelayTransactor // Write-only binding to the contract
	CelestiaRelayFilterer   // Log filterer for contract events
}

// CelestiaRelayCaller is an auto generated read-only Go binding around an Ethereum contract.
type CelestiaRelayCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CelestiaRelayTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CelestiaRelayTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CelestiaRelayFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CelestiaRelayFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CelestiaRelaySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CelestiaRelaySession struct {
	Contract     *CelestiaRelay    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CelestiaRelayCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CelestiaRelayCallerSession struct {
	Contract *CelestiaRelayCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// CelestiaRelayTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CelestiaRelayTransactorSession struct {
	Contract     *CelestiaRelayTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// CelestiaRelayRaw is an auto generated low-level Go binding around an Ethereum contract.
type CelestiaRelayRaw struct {
	Contract *CelestiaRelay // Generic contract binding to access the raw methods on
}

// CelestiaRelayCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CelestiaRelayCallerRaw struct {
	Contract *CelestiaRelayCaller // Generic read-only contract binding to access the raw methods on
}

// CelestiaRelayTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CelestiaRelayTransactorRaw struct {
	Contract *CelestiaRelayTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCelestiaRelay creates a new instance of CelestiaRelay, bound to a specific deployed contract.
func NewCelestiaRelay(address common.Address, backend bind.ContractBackend) (*CelestiaRelay, error) {
	contract, err := bindCelestiaRelay(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CelestiaRelay{CelestiaRelayCaller: CelestiaRelayCaller{contract: contract}, CelestiaRelayTransactor: CelestiaRelayTransactor{contract: contract}, CelestiaRelayFilterer: CelestiaRelayFilterer{contract: contract}}, nil
}

// NewCelestiaRelayCaller creates a new read-only instance of CelestiaRelay, bound to a specific deployed contract.
func NewCelestiaRelayCaller(address common.Address, caller bind.ContractCaller) (*CelestiaRelayCaller, error) {
	contract, err := bindCelestiaRelay(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CelestiaRelayCaller{contract: contract}, nil
}

// NewCelestiaRelayTransactor creates a new write-only instance of CelestiaRelay, bound to a specific deployed contract.
func NewCelestiaRelayTransactor(address common.Address, transactor bind.ContractTransactor) (*CelestiaRelayTransactor, error) {
	contract, err := bindCelestiaRelay(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CelestiaRelayTransactor{contract: contract}, nil
}

// NewCelestiaRelayFilterer creates a new log filterer instance of CelestiaRelay, bound to a specific deployed contract.
func NewCelestiaRelayFilterer(address common.Address, filterer bind.ContractFilterer) (*CelestiaRelayFilterer, error) {
	contract, err := bindCelestiaRelay(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CelestiaRelayFilterer{contract: contract}, nil
}

// bindCelestiaRelay binds a generic wrapper to an already deployed contract.
func bindCelestiaRelay(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CelestiaRelayMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CelestiaRelay *CelestiaRelayRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CelestiaRelay.Contract.CelestiaRelayCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CelestiaRelay *CelestiaRelayRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CelestiaRelay.Contract.CelestiaRelayTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CelestiaRelay *CelestiaRelayRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CelestiaRelay.Contract.CelestiaRelayTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CelestiaRelay *CelestiaRelayCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CelestiaRelay.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CelestiaRelay *CelestiaRelayTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CelestiaRelay.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CelestiaRelay *CelestiaRelayTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CelestiaRelay.Contract.contract.Transact(opts, method, params...)
}

// RelayShares is a paid mutator transaction binding the contract method 0x1ffcf1a9.
//
// Solidity: function relayShares(address _dapp, (bytes[],(uint256,uint256,((bytes1,bytes28),(bytes1,bytes28),bytes32)[])[],(bytes1,bytes28),((bytes1,bytes28),(bytes1,bytes28),bytes32)[],(bytes32[],uint256,uint256)[],(uint256,(uint256,bytes32),(bytes32[],uint256,uint256))) _proof, bytes32 _root) returns(bytes32)
func (_CelestiaRelay *CelestiaRelayTransactor) RelayShares(opts *bind.TransactOpts, _dapp common.Address, _proof SharesProof, _root [32]byte) (*types.Transaction, error) {
	return _CelestiaRelay.contract.Transact(opts, "relayShares", _dapp, _proof, _root)
}

// RelayShares is a paid mutator transaction binding the contract method 0x1ffcf1a9.
//
// Solidity: function relayShares(address _dapp, (bytes[],(uint256,uint256,((bytes1,bytes28),(bytes1,bytes28),bytes32)[])[],(bytes1,bytes28),((bytes1,bytes28),(bytes1,bytes28),bytes32)[],(bytes32[],uint256,uint256)[],(uint256,(uint256,bytes32),(bytes32[],uint256,uint256))) _proof, bytes32 _root) returns(bytes32)
func (_CelestiaRelay *CelestiaRelaySession) RelayShares(_dapp common.Address, _proof SharesProof, _root [32]byte) (*types.Transaction, error) {
	return _CelestiaRelay.Contract.RelayShares(&_CelestiaRelay.TransactOpts, _dapp, _proof, _root)
}

// RelayShares is a paid mutator transaction binding the contract method 0x1ffcf1a9.
//
// Solidity: function relayShares(address _dapp, (bytes[],(uint256,uint256,((bytes1,bytes28),(bytes1,bytes28),bytes32)[])[],(bytes1,bytes28),((bytes1,bytes28),(bytes1,bytes28),bytes32)[],(bytes32[],uint256,uint256)[],(uint256,(uint256,bytes32),(bytes32[],uint256,uint256))) _proof, bytes32 _root) returns(bytes32)
func (_CelestiaRelay *CelestiaRelayTransactorSession) RelayShares(_dapp common.Address, _proof SharesProof, _root [32]byte) (*types.Transaction, error) {
	return _CelestiaRelay.Contract.RelayShares(&_CelestiaRelay.TransactOpts, _dapp, _proof, _root)
}
