// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package shadowvaultcontract

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

// ShadowvaultcontractMetaData contains all meta data concerning the Shadowvaultcontract contract.
var ShadowvaultcontractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"genesisTime\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"balanceof\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"slotNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"proposerIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"parentRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"bodyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fieldIndex\",\"type\":\"uint256\"}],\"name\":\"generateHeaderProof\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"time\",\"type\":\"uint256\"}],\"name\":\"getBeaconRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"slot\",\"type\":\"uint256\"}],\"name\":\"getBeaconRootBySlot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGenesisTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"slotTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"slotNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"shadowWithdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
}

// ShadowvaultcontractABI is the input ABI used to generate the binding from.
// Deprecated: Use ShadowvaultcontractMetaData.ABI instead.
var ShadowvaultcontractABI = ShadowvaultcontractMetaData.ABI

// Shadowvaultcontract is an auto generated Go binding around an Ethereum contract.
type Shadowvaultcontract struct {
	ShadowvaultcontractCaller     // Read-only binding to the contract
	ShadowvaultcontractTransactor // Write-only binding to the contract
	ShadowvaultcontractFilterer   // Log filterer for contract events
}

// ShadowvaultcontractCaller is an auto generated read-only Go binding around an Ethereum contract.
type ShadowvaultcontractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ShadowvaultcontractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ShadowvaultcontractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ShadowvaultcontractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ShadowvaultcontractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ShadowvaultcontractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ShadowvaultcontractSession struct {
	Contract     *Shadowvaultcontract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ShadowvaultcontractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ShadowvaultcontractCallerSession struct {
	Contract *ShadowvaultcontractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// ShadowvaultcontractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ShadowvaultcontractTransactorSession struct {
	Contract     *ShadowvaultcontractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// ShadowvaultcontractRaw is an auto generated low-level Go binding around an Ethereum contract.
type ShadowvaultcontractRaw struct {
	Contract *Shadowvaultcontract // Generic contract binding to access the raw methods on
}

// ShadowvaultcontractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ShadowvaultcontractCallerRaw struct {
	Contract *ShadowvaultcontractCaller // Generic read-only contract binding to access the raw methods on
}

// ShadowvaultcontractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ShadowvaultcontractTransactorRaw struct {
	Contract *ShadowvaultcontractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewShadowvaultcontract creates a new instance of Shadowvaultcontract, bound to a specific deployed contract.
func NewShadowvaultcontract(address common.Address, backend bind.ContractBackend) (*Shadowvaultcontract, error) {
	contract, err := bindShadowvaultcontract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Shadowvaultcontract{ShadowvaultcontractCaller: ShadowvaultcontractCaller{contract: contract}, ShadowvaultcontractTransactor: ShadowvaultcontractTransactor{contract: contract}, ShadowvaultcontractFilterer: ShadowvaultcontractFilterer{contract: contract}}, nil
}

// NewShadowvaultcontractCaller creates a new read-only instance of Shadowvaultcontract, bound to a specific deployed contract.
func NewShadowvaultcontractCaller(address common.Address, caller bind.ContractCaller) (*ShadowvaultcontractCaller, error) {
	contract, err := bindShadowvaultcontract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ShadowvaultcontractCaller{contract: contract}, nil
}

// NewShadowvaultcontractTransactor creates a new write-only instance of Shadowvaultcontract, bound to a specific deployed contract.
func NewShadowvaultcontractTransactor(address common.Address, transactor bind.ContractTransactor) (*ShadowvaultcontractTransactor, error) {
	contract, err := bindShadowvaultcontract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ShadowvaultcontractTransactor{contract: contract}, nil
}

// NewShadowvaultcontractFilterer creates a new log filterer instance of Shadowvaultcontract, bound to a specific deployed contract.
func NewShadowvaultcontractFilterer(address common.Address, filterer bind.ContractFilterer) (*ShadowvaultcontractFilterer, error) {
	contract, err := bindShadowvaultcontract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ShadowvaultcontractFilterer{contract: contract}, nil
}

// bindShadowvaultcontract binds a generic wrapper to an already deployed contract.
func bindShadowvaultcontract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ShadowvaultcontractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Shadowvaultcontract *ShadowvaultcontractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Shadowvaultcontract.Contract.ShadowvaultcontractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Shadowvaultcontract *ShadowvaultcontractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.ShadowvaultcontractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Shadowvaultcontract *ShadowvaultcontractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.ShadowvaultcontractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Shadowvaultcontract *ShadowvaultcontractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Shadowvaultcontract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Shadowvaultcontract *ShadowvaultcontractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Shadowvaultcontract *ShadowvaultcontractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.contract.Transact(opts, method, params...)
}

// Balanceof is a free data retrieval call binding the contract method 0x3d64125b.
//
// Solidity: function balanceof(address addr) view returns(uint256)
func (_Shadowvaultcontract *ShadowvaultcontractCaller) Balanceof(opts *bind.CallOpts, addr common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Shadowvaultcontract.contract.Call(opts, &out, "balanceof", addr)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Balanceof is a free data retrieval call binding the contract method 0x3d64125b.
//
// Solidity: function balanceof(address addr) view returns(uint256)
func (_Shadowvaultcontract *ShadowvaultcontractSession) Balanceof(addr common.Address) (*big.Int, error) {
	return _Shadowvaultcontract.Contract.Balanceof(&_Shadowvaultcontract.CallOpts, addr)
}

// Balanceof is a free data retrieval call binding the contract method 0x3d64125b.
//
// Solidity: function balanceof(address addr) view returns(uint256)
func (_Shadowvaultcontract *ShadowvaultcontractCallerSession) Balanceof(addr common.Address) (*big.Int, error) {
	return _Shadowvaultcontract.Contract.Balanceof(&_Shadowvaultcontract.CallOpts, addr)
}

// GenerateHeaderProof is a free data retrieval call binding the contract method 0x68a6ad99.
//
// Solidity: function generateHeaderProof(uint256 slotNumber, uint256 proposerIndex, bytes32 parentRoot, bytes32 stateRoot, bytes32 bodyRoot, uint256 fieldIndex) view returns(bytes)
func (_Shadowvaultcontract *ShadowvaultcontractCaller) GenerateHeaderProof(opts *bind.CallOpts, slotNumber *big.Int, proposerIndex *big.Int, parentRoot [32]byte, stateRoot [32]byte, bodyRoot [32]byte, fieldIndex *big.Int) ([]byte, error) {
	var out []interface{}
	err := _Shadowvaultcontract.contract.Call(opts, &out, "generateHeaderProof", slotNumber, proposerIndex, parentRoot, stateRoot, bodyRoot, fieldIndex)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GenerateHeaderProof is a free data retrieval call binding the contract method 0x68a6ad99.
//
// Solidity: function generateHeaderProof(uint256 slotNumber, uint256 proposerIndex, bytes32 parentRoot, bytes32 stateRoot, bytes32 bodyRoot, uint256 fieldIndex) view returns(bytes)
func (_Shadowvaultcontract *ShadowvaultcontractSession) GenerateHeaderProof(slotNumber *big.Int, proposerIndex *big.Int, parentRoot [32]byte, stateRoot [32]byte, bodyRoot [32]byte, fieldIndex *big.Int) ([]byte, error) {
	return _Shadowvaultcontract.Contract.GenerateHeaderProof(&_Shadowvaultcontract.CallOpts, slotNumber, proposerIndex, parentRoot, stateRoot, bodyRoot, fieldIndex)
}

// GenerateHeaderProof is a free data retrieval call binding the contract method 0x68a6ad99.
//
// Solidity: function generateHeaderProof(uint256 slotNumber, uint256 proposerIndex, bytes32 parentRoot, bytes32 stateRoot, bytes32 bodyRoot, uint256 fieldIndex) view returns(bytes)
func (_Shadowvaultcontract *ShadowvaultcontractCallerSession) GenerateHeaderProof(slotNumber *big.Int, proposerIndex *big.Int, parentRoot [32]byte, stateRoot [32]byte, bodyRoot [32]byte, fieldIndex *big.Int) ([]byte, error) {
	return _Shadowvaultcontract.Contract.GenerateHeaderProof(&_Shadowvaultcontract.CallOpts, slotNumber, proposerIndex, parentRoot, stateRoot, bodyRoot, fieldIndex)
}

// GetBeaconRoot is a free data retrieval call binding the contract method 0x661a052f.
//
// Solidity: function getBeaconRoot(uint256 time) view returns(bytes32)
func (_Shadowvaultcontract *ShadowvaultcontractCaller) GetBeaconRoot(opts *bind.CallOpts, time *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Shadowvaultcontract.contract.Call(opts, &out, "getBeaconRoot", time)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBeaconRoot is a free data retrieval call binding the contract method 0x661a052f.
//
// Solidity: function getBeaconRoot(uint256 time) view returns(bytes32)
func (_Shadowvaultcontract *ShadowvaultcontractSession) GetBeaconRoot(time *big.Int) ([32]byte, error) {
	return _Shadowvaultcontract.Contract.GetBeaconRoot(&_Shadowvaultcontract.CallOpts, time)
}

// GetBeaconRoot is a free data retrieval call binding the contract method 0x661a052f.
//
// Solidity: function getBeaconRoot(uint256 time) view returns(bytes32)
func (_Shadowvaultcontract *ShadowvaultcontractCallerSession) GetBeaconRoot(time *big.Int) ([32]byte, error) {
	return _Shadowvaultcontract.Contract.GetBeaconRoot(&_Shadowvaultcontract.CallOpts, time)
}

// GetBeaconRootBySlot is a free data retrieval call binding the contract method 0x692d1ddf.
//
// Solidity: function getBeaconRootBySlot(uint256 slot) view returns(bytes32)
func (_Shadowvaultcontract *ShadowvaultcontractCaller) GetBeaconRootBySlot(opts *bind.CallOpts, slot *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Shadowvaultcontract.contract.Call(opts, &out, "getBeaconRootBySlot", slot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBeaconRootBySlot is a free data retrieval call binding the contract method 0x692d1ddf.
//
// Solidity: function getBeaconRootBySlot(uint256 slot) view returns(bytes32)
func (_Shadowvaultcontract *ShadowvaultcontractSession) GetBeaconRootBySlot(slot *big.Int) ([32]byte, error) {
	return _Shadowvaultcontract.Contract.GetBeaconRootBySlot(&_Shadowvaultcontract.CallOpts, slot)
}

// GetBeaconRootBySlot is a free data retrieval call binding the contract method 0x692d1ddf.
//
// Solidity: function getBeaconRootBySlot(uint256 slot) view returns(bytes32)
func (_Shadowvaultcontract *ShadowvaultcontractCallerSession) GetBeaconRootBySlot(slot *big.Int) ([32]byte, error) {
	return _Shadowvaultcontract.Contract.GetBeaconRootBySlot(&_Shadowvaultcontract.CallOpts, slot)
}

// GetGenesisTime is a free data retrieval call binding the contract method 0x723d8e96.
//
// Solidity: function getGenesisTime() view returns(uint256)
func (_Shadowvaultcontract *ShadowvaultcontractCaller) GetGenesisTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Shadowvaultcontract.contract.Call(opts, &out, "getGenesisTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetGenesisTime is a free data retrieval call binding the contract method 0x723d8e96.
//
// Solidity: function getGenesisTime() view returns(uint256)
func (_Shadowvaultcontract *ShadowvaultcontractSession) GetGenesisTime() (*big.Int, error) {
	return _Shadowvaultcontract.Contract.GetGenesisTime(&_Shadowvaultcontract.CallOpts)
}

// GetGenesisTime is a free data retrieval call binding the contract method 0x723d8e96.
//
// Solidity: function getGenesisTime() view returns(uint256)
func (_Shadowvaultcontract *ShadowvaultcontractCallerSession) GetGenesisTime() (*big.Int, error) {
	return _Shadowvaultcontract.Contract.GetGenesisTime(&_Shadowvaultcontract.CallOpts)
}

// ShadowWithdraw is a paid mutator transaction binding the contract method 0x7c340fcf.
//
// Solidity: function shadowWithdraw(uint256 slotTime, uint256 slotNumber, bytes proof, address target, uint256 amount) returns()
func (_Shadowvaultcontract *ShadowvaultcontractTransactor) ShadowWithdraw(opts *bind.TransactOpts, slotTime *big.Int, slotNumber *big.Int, proof []byte, target common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Shadowvaultcontract.contract.Transact(opts, "shadowWithdraw", slotTime, slotNumber, proof, target, amount)
}

// ShadowWithdraw is a paid mutator transaction binding the contract method 0x7c340fcf.
//
// Solidity: function shadowWithdraw(uint256 slotTime, uint256 slotNumber, bytes proof, address target, uint256 amount) returns()
func (_Shadowvaultcontract *ShadowvaultcontractSession) ShadowWithdraw(slotTime *big.Int, slotNumber *big.Int, proof []byte, target common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.ShadowWithdraw(&_Shadowvaultcontract.TransactOpts, slotTime, slotNumber, proof, target, amount)
}

// ShadowWithdraw is a paid mutator transaction binding the contract method 0x7c340fcf.
//
// Solidity: function shadowWithdraw(uint256 slotTime, uint256 slotNumber, bytes proof, address target, uint256 amount) returns()
func (_Shadowvaultcontract *ShadowvaultcontractTransactorSession) ShadowWithdraw(slotTime *big.Int, slotNumber *big.Int, proof []byte, target common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.ShadowWithdraw(&_Shadowvaultcontract.TransactOpts, slotTime, slotNumber, proof, target, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 amount) returns()
func (_Shadowvaultcontract *ShadowvaultcontractTransactor) Withdraw(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _Shadowvaultcontract.contract.Transact(opts, "withdraw", amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 amount) returns()
func (_Shadowvaultcontract *ShadowvaultcontractSession) Withdraw(amount *big.Int) (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.Withdraw(&_Shadowvaultcontract.TransactOpts, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 amount) returns()
func (_Shadowvaultcontract *ShadowvaultcontractTransactorSession) Withdraw(amount *big.Int) (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.Withdraw(&_Shadowvaultcontract.TransactOpts, amount)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Shadowvaultcontract *ShadowvaultcontractTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Shadowvaultcontract.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Shadowvaultcontract *ShadowvaultcontractSession) Receive() (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.Receive(&_Shadowvaultcontract.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Shadowvaultcontract *ShadowvaultcontractTransactorSession) Receive() (*types.Transaction, error) {
	return _Shadowvaultcontract.Contract.Receive(&_Shadowvaultcontract.TransactOpts)
}
