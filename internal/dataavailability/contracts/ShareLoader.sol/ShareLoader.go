// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package shareloader

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

// ShareloaderMetaData contains all meta data concerning the Shareloader contract.
var ShareloaderMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_blobstreamX\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"blobstreamX\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIDAOracle\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"verifyShares\",\"inputs\":[{\"name\":\"_proof\",\"type\":\"tuple\",\"internalType\":\"structSharesProof\",\"components\":[{\"name\":\"data\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"shareProofs\",\"type\":\"tuple[]\",\"internalType\":\"structNamespaceMerkleMultiproof[]\",\"components\":[{\"name\":\"beginKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"endKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"sideNodes\",\"type\":\"tuple[]\",\"internalType\":\"structNamespaceNode[]\",\"components\":[{\"name\":\"min\",\"type\":\"tuple\",\"internalType\":\"structNamespace\",\"components\":[{\"name\":\"version\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"id\",\"type\":\"bytes28\",\"internalType\":\"bytes28\"}]},{\"name\":\"max\",\"type\":\"tuple\",\"internalType\":\"structNamespace\",\"components\":[{\"name\":\"version\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"id\",\"type\":\"bytes28\",\"internalType\":\"bytes28\"}]},{\"name\":\"digest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]}]},{\"name\":\"namespace\",\"type\":\"tuple\",\"internalType\":\"structNamespace\",\"components\":[{\"name\":\"version\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"id\",\"type\":\"bytes28\",\"internalType\":\"bytes28\"}]},{\"name\":\"rowRoots\",\"type\":\"tuple[]\",\"internalType\":\"structNamespaceNode[]\",\"components\":[{\"name\":\"min\",\"type\":\"tuple\",\"internalType\":\"structNamespace\",\"components\":[{\"name\":\"version\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"id\",\"type\":\"bytes28\",\"internalType\":\"bytes28\"}]},{\"name\":\"max\",\"type\":\"tuple\",\"internalType\":\"structNamespace\",\"components\":[{\"name\":\"version\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"id\",\"type\":\"bytes28\",\"internalType\":\"bytes28\"}]},{\"name\":\"digest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"rowProofs\",\"type\":\"tuple[]\",\"internalType\":\"structBinaryMerkleProof[]\",\"components\":[{\"name\":\"sideNodes\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"},{\"name\":\"key\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"numLeaves\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"attestationProof\",\"type\":\"tuple\",\"internalType\":\"structAttestationProof\",\"components\":[{\"name\":\"tupleRootNonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"tuple\",\"type\":\"tuple\",\"internalType\":\"structDataRootTuple\",\"components\":[{\"name\":\"height\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"dataRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"proof\",\"type\":\"tuple\",\"internalType\":\"structBinaryMerkleProof\",\"components\":[{\"name\":\"sideNodes\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"},{\"name\":\"key\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"numLeaves\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]}]},{\"name\":\"_root\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumDAVerifier.ErrorCodes\"}],\"stateMutability\":\"view\"}]",
}

// ShareloaderABI is the input ABI used to generate the binding from.
// Deprecated: Use ShareloaderMetaData.ABI instead.
var ShareloaderABI = ShareloaderMetaData.ABI

// Shareloader is an auto generated Go binding around an Ethereum contract.
type Shareloader struct {
	ShareloaderCaller     // Read-only binding to the contract
	ShareloaderTransactor // Write-only binding to the contract
	ShareloaderFilterer   // Log filterer for contract events
}

// ShareloaderCaller is an auto generated read-only Go binding around an Ethereum contract.
type ShareloaderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ShareloaderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ShareloaderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ShareloaderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ShareloaderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ShareloaderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ShareloaderSession struct {
	Contract     *Shareloader      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ShareloaderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ShareloaderCallerSession struct {
	Contract *ShareloaderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// ShareloaderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ShareloaderTransactorSession struct {
	Contract     *ShareloaderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// ShareloaderRaw is an auto generated low-level Go binding around an Ethereum contract.
type ShareloaderRaw struct {
	Contract *Shareloader // Generic contract binding to access the raw methods on
}

// ShareloaderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ShareloaderCallerRaw struct {
	Contract *ShareloaderCaller // Generic read-only contract binding to access the raw methods on
}

// ShareloaderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ShareloaderTransactorRaw struct {
	Contract *ShareloaderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewShareloader creates a new instance of Shareloader, bound to a specific deployed contract.
func NewShareloader(address common.Address, backend bind.ContractBackend) (*Shareloader, error) {
	contract, err := bindShareloader(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Shareloader{ShareloaderCaller: ShareloaderCaller{contract: contract}, ShareloaderTransactor: ShareloaderTransactor{contract: contract}, ShareloaderFilterer: ShareloaderFilterer{contract: contract}}, nil
}

// NewShareloaderCaller creates a new read-only instance of Shareloader, bound to a specific deployed contract.
func NewShareloaderCaller(address common.Address, caller bind.ContractCaller) (*ShareloaderCaller, error) {
	contract, err := bindShareloader(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ShareloaderCaller{contract: contract}, nil
}

// NewShareloaderTransactor creates a new write-only instance of Shareloader, bound to a specific deployed contract.
func NewShareloaderTransactor(address common.Address, transactor bind.ContractTransactor) (*ShareloaderTransactor, error) {
	contract, err := bindShareloader(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ShareloaderTransactor{contract: contract}, nil
}

// NewShareloaderFilterer creates a new log filterer instance of Shareloader, bound to a specific deployed contract.
func NewShareloaderFilterer(address common.Address, filterer bind.ContractFilterer) (*ShareloaderFilterer, error) {
	contract, err := bindShareloader(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ShareloaderFilterer{contract: contract}, nil
}

// bindShareloader binds a generic wrapper to an already deployed contract.
func bindShareloader(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ShareloaderMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Shareloader *ShareloaderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Shareloader.Contract.ShareloaderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Shareloader *ShareloaderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Shareloader.Contract.ShareloaderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Shareloader *ShareloaderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Shareloader.Contract.ShareloaderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Shareloader *ShareloaderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Shareloader.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Shareloader *ShareloaderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Shareloader.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Shareloader *ShareloaderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Shareloader.Contract.contract.Transact(opts, method, params...)
}

// BlobstreamX is a free data retrieval call binding the contract method 0xd020358f.
//
// Solidity: function blobstreamX() view returns(address)
func (_Shareloader *ShareloaderCaller) BlobstreamX(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Shareloader.contract.Call(opts, &out, "blobstreamX")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BlobstreamX is a free data retrieval call binding the contract method 0xd020358f.
//
// Solidity: function blobstreamX() view returns(address)
func (_Shareloader *ShareloaderSession) BlobstreamX() (common.Address, error) {
	return _Shareloader.Contract.BlobstreamX(&_Shareloader.CallOpts)
}

// BlobstreamX is a free data retrieval call binding the contract method 0xd020358f.
//
// Solidity: function blobstreamX() view returns(address)
func (_Shareloader *ShareloaderCallerSession) BlobstreamX() (common.Address, error) {
	return _Shareloader.Contract.BlobstreamX(&_Shareloader.CallOpts)
}

// VerifyShares is a free data retrieval call binding the contract method 0x996f0f1f.
//
// Solidity: function verifyShares((bytes[],(uint256,uint256,((bytes1,bytes28),(bytes1,bytes28),bytes32)[])[],(bytes1,bytes28),((bytes1,bytes28),(bytes1,bytes28),bytes32)[],(bytes32[],uint256,uint256)[],(uint256,(uint256,bytes32),(bytes32[],uint256,uint256))) _proof, bytes32 _root) view returns(bool, uint8)
func (_Shareloader *ShareloaderCaller) VerifyShares(opts *bind.CallOpts, _proof SharesProof, _root [32]byte) (bool, uint8, error) {
	var out []interface{}
	err := _Shareloader.contract.Call(opts, &out, "verifyShares", _proof, _root)

	if err != nil {
		return *new(bool), *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new(uint8)).(*uint8)

	return out0, out1, err

}

// VerifyShares is a free data retrieval call binding the contract method 0x996f0f1f.
//
// Solidity: function verifyShares((bytes[],(uint256,uint256,((bytes1,bytes28),(bytes1,bytes28),bytes32)[])[],(bytes1,bytes28),((bytes1,bytes28),(bytes1,bytes28),bytes32)[],(bytes32[],uint256,uint256)[],(uint256,(uint256,bytes32),(bytes32[],uint256,uint256))) _proof, bytes32 _root) view returns(bool, uint8)
func (_Shareloader *ShareloaderSession) VerifyShares(_proof SharesProof, _root [32]byte) (bool, uint8, error) {
	return _Shareloader.Contract.VerifyShares(&_Shareloader.CallOpts, _proof, _root)
}

// VerifyShares is a free data retrieval call binding the contract method 0x996f0f1f.
//
// Solidity: function verifyShares((bytes[],(uint256,uint256,((bytes1,bytes28),(bytes1,bytes28),bytes32)[])[],(bytes1,bytes28),((bytes1,bytes28),(bytes1,bytes28),bytes32)[],(bytes32[],uint256,uint256)[],(uint256,(uint256,bytes32),(bytes32[],uint256,uint256))) _proof, bytes32 _root) view returns(bool, uint8)
func (_Shareloader *ShareloaderCallerSession) VerifyShares(_proof SharesProof, _root [32]byte) (bool, uint8, error) {
	return _Shareloader.Contract.VerifyShares(&_Shareloader.CallOpts, _proof, _root)
}
