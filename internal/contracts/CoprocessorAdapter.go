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

// CoprocessorAdapterMetaData contains all meta data concerning the CoprocessorAdapter contract.
var CoprocessorAdapterMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"output\",\"type\":\"bytes\"}],\"name\":\"ResultReceived\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"notice\",\"type\":\"bytes\"}],\"name\":\"handleNotice\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nftContract\",\"outputs\":[{\"internalType\":\"contractINFTPlayers\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"payloadHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"notice\",\"type\":\"bytes\"}],\"name\":\"nonodoHandleNotice\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"nftAddress\",\"type\":\"address\"}],\"name\":\"setNFTPlayersContract\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// CoprocessorAdapterABI is the input ABI used to generate the binding from.
// Deprecated: Use CoprocessorAdapterMetaData.ABI instead.
var CoprocessorAdapterABI = CoprocessorAdapterMetaData.ABI

// CoprocessorAdapter is an auto generated Go binding around an Ethereum contract.
type CoprocessorAdapter struct {
	CoprocessorAdapterCaller     // Read-only binding to the contract
	CoprocessorAdapterTransactor // Write-only binding to the contract
	CoprocessorAdapterFilterer   // Log filterer for contract events
}

// CoprocessorAdapterCaller is an auto generated read-only Go binding around an Ethereum contract.
type CoprocessorAdapterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CoprocessorAdapterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CoprocessorAdapterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CoprocessorAdapterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CoprocessorAdapterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CoprocessorAdapterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CoprocessorAdapterSession struct {
	Contract     *CoprocessorAdapter // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// CoprocessorAdapterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CoprocessorAdapterCallerSession struct {
	Contract *CoprocessorAdapterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// CoprocessorAdapterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CoprocessorAdapterTransactorSession struct {
	Contract     *CoprocessorAdapterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// CoprocessorAdapterRaw is an auto generated low-level Go binding around an Ethereum contract.
type CoprocessorAdapterRaw struct {
	Contract *CoprocessorAdapter // Generic contract binding to access the raw methods on
}

// CoprocessorAdapterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CoprocessorAdapterCallerRaw struct {
	Contract *CoprocessorAdapterCaller // Generic read-only contract binding to access the raw methods on
}

// CoprocessorAdapterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CoprocessorAdapterTransactorRaw struct {
	Contract *CoprocessorAdapterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCoprocessorAdapter creates a new instance of CoprocessorAdapter, bound to a specific deployed contract.
func NewCoprocessorAdapter(address common.Address, backend bind.ContractBackend) (*CoprocessorAdapter, error) {
	contract, err := bindCoprocessorAdapter(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CoprocessorAdapter{CoprocessorAdapterCaller: CoprocessorAdapterCaller{contract: contract}, CoprocessorAdapterTransactor: CoprocessorAdapterTransactor{contract: contract}, CoprocessorAdapterFilterer: CoprocessorAdapterFilterer{contract: contract}}, nil
}

// NewCoprocessorAdapterCaller creates a new read-only instance of CoprocessorAdapter, bound to a specific deployed contract.
func NewCoprocessorAdapterCaller(address common.Address, caller bind.ContractCaller) (*CoprocessorAdapterCaller, error) {
	contract, err := bindCoprocessorAdapter(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CoprocessorAdapterCaller{contract: contract}, nil
}

// NewCoprocessorAdapterTransactor creates a new write-only instance of CoprocessorAdapter, bound to a specific deployed contract.
func NewCoprocessorAdapterTransactor(address common.Address, transactor bind.ContractTransactor) (*CoprocessorAdapterTransactor, error) {
	contract, err := bindCoprocessorAdapter(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CoprocessorAdapterTransactor{contract: contract}, nil
}

// NewCoprocessorAdapterFilterer creates a new log filterer instance of CoprocessorAdapter, bound to a specific deployed contract.
func NewCoprocessorAdapterFilterer(address common.Address, filterer bind.ContractFilterer) (*CoprocessorAdapterFilterer, error) {
	contract, err := bindCoprocessorAdapter(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CoprocessorAdapterFilterer{contract: contract}, nil
}

// bindCoprocessorAdapter binds a generic wrapper to an already deployed contract.
func bindCoprocessorAdapter(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CoprocessorAdapterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CoprocessorAdapter *CoprocessorAdapterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CoprocessorAdapter.Contract.CoprocessorAdapterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CoprocessorAdapter *CoprocessorAdapterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.CoprocessorAdapterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CoprocessorAdapter *CoprocessorAdapterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.CoprocessorAdapterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CoprocessorAdapter *CoprocessorAdapterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CoprocessorAdapter.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CoprocessorAdapter *CoprocessorAdapterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CoprocessorAdapter *CoprocessorAdapterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.contract.Transact(opts, method, params...)
}

// NftContract is a free data retrieval call binding the contract method 0xd56d229d.
//
// Solidity: function nftContract() view returns(address)
func (_CoprocessorAdapter *CoprocessorAdapterCaller) NftContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CoprocessorAdapter.contract.Call(opts, &out, "nftContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NftContract is a free data retrieval call binding the contract method 0xd56d229d.
//
// Solidity: function nftContract() view returns(address)
func (_CoprocessorAdapter *CoprocessorAdapterSession) NftContract() (common.Address, error) {
	return _CoprocessorAdapter.Contract.NftContract(&_CoprocessorAdapter.CallOpts)
}

// NftContract is a free data retrieval call binding the contract method 0xd56d229d.
//
// Solidity: function nftContract() view returns(address)
func (_CoprocessorAdapter *CoprocessorAdapterCallerSession) NftContract() (common.Address, error) {
	return _CoprocessorAdapter.Contract.NftContract(&_CoprocessorAdapter.CallOpts)
}

// HandleNotice is a paid mutator transaction binding the contract method 0xa3bc9466.
//
// Solidity: function handleNotice(bytes32 payloadHash, bytes notice) returns()
func (_CoprocessorAdapter *CoprocessorAdapterTransactor) HandleNotice(opts *bind.TransactOpts, payloadHash [32]byte, notice []byte) (*types.Transaction, error) {
	return _CoprocessorAdapter.contract.Transact(opts, "handleNotice", payloadHash, notice)
}

// HandleNotice is a paid mutator transaction binding the contract method 0xa3bc9466.
//
// Solidity: function handleNotice(bytes32 payloadHash, bytes notice) returns()
func (_CoprocessorAdapter *CoprocessorAdapterSession) HandleNotice(payloadHash [32]byte, notice []byte) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.HandleNotice(&_CoprocessorAdapter.TransactOpts, payloadHash, notice)
}

// HandleNotice is a paid mutator transaction binding the contract method 0xa3bc9466.
//
// Solidity: function handleNotice(bytes32 payloadHash, bytes notice) returns()
func (_CoprocessorAdapter *CoprocessorAdapterTransactorSession) HandleNotice(payloadHash [32]byte, notice []byte) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.HandleNotice(&_CoprocessorAdapter.TransactOpts, payloadHash, notice)
}

// NonodoHandleNotice is a paid mutator transaction binding the contract method 0xa77d99c8.
//
// Solidity: function nonodoHandleNotice(bytes32 payloadHash, bytes notice) returns()
func (_CoprocessorAdapter *CoprocessorAdapterTransactor) NonodoHandleNotice(opts *bind.TransactOpts, payloadHash [32]byte, notice []byte) (*types.Transaction, error) {
	return _CoprocessorAdapter.contract.Transact(opts, "nonodoHandleNotice", payloadHash, notice)
}

// NonodoHandleNotice is a paid mutator transaction binding the contract method 0xa77d99c8.
//
// Solidity: function nonodoHandleNotice(bytes32 payloadHash, bytes notice) returns()
func (_CoprocessorAdapter *CoprocessorAdapterSession) NonodoHandleNotice(payloadHash [32]byte, notice []byte) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.NonodoHandleNotice(&_CoprocessorAdapter.TransactOpts, payloadHash, notice)
}

// NonodoHandleNotice is a paid mutator transaction binding the contract method 0xa77d99c8.
//
// Solidity: function nonodoHandleNotice(bytes32 payloadHash, bytes notice) returns()
func (_CoprocessorAdapter *CoprocessorAdapterTransactorSession) NonodoHandleNotice(payloadHash [32]byte, notice []byte) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.NonodoHandleNotice(&_CoprocessorAdapter.TransactOpts, payloadHash, notice)
}

// SetNFTPlayersContract is a paid mutator transaction binding the contract method 0x06553cf3.
//
// Solidity: function setNFTPlayersContract(address nftAddress) returns()
func (_CoprocessorAdapter *CoprocessorAdapterTransactor) SetNFTPlayersContract(opts *bind.TransactOpts, nftAddress common.Address) (*types.Transaction, error) {
	return _CoprocessorAdapter.contract.Transact(opts, "setNFTPlayersContract", nftAddress)
}

// SetNFTPlayersContract is a paid mutator transaction binding the contract method 0x06553cf3.
//
// Solidity: function setNFTPlayersContract(address nftAddress) returns()
func (_CoprocessorAdapter *CoprocessorAdapterSession) SetNFTPlayersContract(nftAddress common.Address) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.SetNFTPlayersContract(&_CoprocessorAdapter.TransactOpts, nftAddress)
}

// SetNFTPlayersContract is a paid mutator transaction binding the contract method 0x06553cf3.
//
// Solidity: function setNFTPlayersContract(address nftAddress) returns()
func (_CoprocessorAdapter *CoprocessorAdapterTransactorSession) SetNFTPlayersContract(nftAddress common.Address) (*types.Transaction, error) {
	return _CoprocessorAdapter.Contract.SetNFTPlayersContract(&_CoprocessorAdapter.TransactOpts, nftAddress)
}

// CoprocessorAdapterResultReceivedIterator is returned from FilterResultReceived and is used to iterate over the raw logs and unpacked data for ResultReceived events raised by the CoprocessorAdapter contract.
type CoprocessorAdapterResultReceivedIterator struct {
	Event *CoprocessorAdapterResultReceived // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *CoprocessorAdapterResultReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CoprocessorAdapterResultReceived)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(CoprocessorAdapterResultReceived)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *CoprocessorAdapterResultReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CoprocessorAdapterResultReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CoprocessorAdapterResultReceived represents a ResultReceived event raised by the CoprocessorAdapter contract.
type CoprocessorAdapterResultReceived struct {
	PayloadHash [32]byte
	Output      []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterResultReceived is a free log retrieval operation binding the contract event 0x11ee75453d1327abdfe4f7a8ad4821587837391f365bef110620811d7a2fa591.
//
// Solidity: event ResultReceived(bytes32 payloadHash, bytes output)
func (_CoprocessorAdapter *CoprocessorAdapterFilterer) FilterResultReceived(opts *bind.FilterOpts) (*CoprocessorAdapterResultReceivedIterator, error) {

	logs, sub, err := _CoprocessorAdapter.contract.FilterLogs(opts, "ResultReceived")
	if err != nil {
		return nil, err
	}
	return &CoprocessorAdapterResultReceivedIterator{contract: _CoprocessorAdapter.contract, event: "ResultReceived", logs: logs, sub: sub}, nil
}

// WatchResultReceived is a free log subscription operation binding the contract event 0x11ee75453d1327abdfe4f7a8ad4821587837391f365bef110620811d7a2fa591.
//
// Solidity: event ResultReceived(bytes32 payloadHash, bytes output)
func (_CoprocessorAdapter *CoprocessorAdapterFilterer) WatchResultReceived(opts *bind.WatchOpts, sink chan<- *CoprocessorAdapterResultReceived) (event.Subscription, error) {

	logs, sub, err := _CoprocessorAdapter.contract.WatchLogs(opts, "ResultReceived")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CoprocessorAdapterResultReceived)
				if err := _CoprocessorAdapter.contract.UnpackLog(event, "ResultReceived", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseResultReceived is a log parse operation binding the contract event 0x11ee75453d1327abdfe4f7a8ad4821587837391f365bef110620811d7a2fa591.
//
// Solidity: event ResultReceived(bytes32 payloadHash, bytes output)
func (_CoprocessorAdapter *CoprocessorAdapterFilterer) ParseResultReceived(log types.Log) (*CoprocessorAdapterResultReceived, error) {
	event := new(CoprocessorAdapterResultReceived)
	if err := _CoprocessorAdapter.contract.UnpackLog(event, "ResultReceived", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
