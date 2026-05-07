// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package batchcontract

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

// BatchDepositContractMetaData contains all meta data concerning the BatchDepositContract contract.
var BatchDepositContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_depositContract\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"pubkeys\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"},{\"internalType\":\"bytes32[]\",\"name\":\"dataRoots\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes\",\"name\":\"withdrawalCredentials\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"amountWei\",\"type\":\"uint256\"}],\"name\":\"batchDeposit\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"depositContract\",\"outputs\":[{\"internalType\":\"contractIDepositContract\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a0346100c957601f6105e038819003918201601f19168301916001600160401b038311848410176100cd578084926020946040528339810103126100c957516001600160a01b038116908190036100c9578015610078576080526040516104fe90816100e28239608051818181604701526101840152f35b60405162461bcd60e51b815260206004820152602360248201527f42617463684465706f7369743a207a65726f206465706f73697420636f6e74726044820152621858dd60ea1b6064820152608490fd5b5f80fd5b634e487b7160e01b5f52604160045260245ffdfe6080806040526004361015610012575f80fd5b5f3560e01c908163ddbd9dd91461007a575063e94ad65b14610032575f80fd5b34610076575f366003190112610076576040517f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168152602090f35b5f80fd5b60a03660031901126100765760043567ffffffffffffffff8111610076576100a6903690600401610462565b9160243567ffffffffffffffff8111610076576100c7903690600401610462565b9390916044359167ffffffffffffffff831161007657366023840112156100765782600401359167ffffffffffffffff8311610076573660248460051b860101116100765760643567ffffffffffffffff81116100765761012c903690600401610462565b9790956084359285156104205750603085028581046030036102eb5784036103dc57606085028581046060036102eb57820361038b5760208903610346578483028381048614841517156102eb5734036103015793967f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031694905f5b898110156102ff57603081029080158183046030148117156102eb5760018201908183116102eb5760308202938215948381046030148617156102eb576101f8918b88610490565b929094606085029285840460601417156102eb576060820291820460601417156102eb57610227918888610490565b9390918a3b15610076576102888f93610264968f610276905f976040519a8b9889986304512a2360e31b8a52608060048b015260848a01916104a8565b878103600319016024890152916104a8565b848103600319016044860152916104a8565b60248d8660051b01013560648301520381898c5af180156102e0576102b2575b60019150016101b0565b67ffffffffffffffff82116102cc576001916040526102a8565b634e487b7160e01b5f52604160045260245ffd5b6040513d5f823e3d90fd5b634e487b7160e01b5f52601160045260245ffd5b005b60405162461bcd60e51b815260206004820152601760248201527f42617463684465706f7369743a206261642076616c75650000000000000000006044820152606490fd5b60405162461bcd60e51b815260206004820152601e60248201527f42617463684465706f7369743a20626164206372656473206c656e67746800006044820152606490fd5b60405162461bcd60e51b815260206004820152602360248201527f42617463684465706f7369743a20626164207369676e617475726573206c656e6044820152620cee8d60eb1b6064820152608490fd5b606460405162461bcd60e51b815260206004820152602060248201527f42617463684465706f7369743a20626164207075626b657973206c656e6774686044820152fd5b62461bcd60e51b815260206004820152601960248201527f42617463684465706f7369743a20656d707479206261746368000000000000006044820152606490fd5b9181601f840112156100765782359167ffffffffffffffff8311610076576020838186019501011161007657565b90939293848311610076578411610076578101920390565b908060209392818452848401375f828201840152601f01601f191601019056fea26469706673582212200fa012285dbbc8ae854030852f9cb7b2f6496cbff2a5e662c11e8aedb1e99e3b64736f6c634300081e0033",
}

// BatchDepositContractABI is the input ABI used to generate the binding from.
// Deprecated: Use BatchDepositContractMetaData.ABI instead.
var BatchDepositContractABI = BatchDepositContractMetaData.ABI

// BatchDepositContractBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BatchDepositContractMetaData.Bin instead.
var BatchDepositContractBin = BatchDepositContractMetaData.Bin

// DeployBatchDepositContract deploys a new Ethereum contract, binding an instance of BatchDepositContract to it.
func DeployBatchDepositContract(auth *bind.TransactOpts, backend bind.ContractBackend, _depositContract common.Address) (common.Address, *types.Transaction, *BatchDepositContract, error) {
	parsed, err := BatchDepositContractMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BatchDepositContractBin), backend, _depositContract)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BatchDepositContract{BatchDepositContractCaller: BatchDepositContractCaller{contract: contract}, BatchDepositContractTransactor: BatchDepositContractTransactor{contract: contract}, BatchDepositContractFilterer: BatchDepositContractFilterer{contract: contract}}, nil
}

// BatchDepositContract is an auto generated Go binding around an Ethereum contract.
type BatchDepositContract struct {
	BatchDepositContractCaller     // Read-only binding to the contract
	BatchDepositContractTransactor // Write-only binding to the contract
	BatchDepositContractFilterer   // Log filterer for contract events
}

// BatchDepositContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type BatchDepositContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BatchDepositContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BatchDepositContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BatchDepositContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BatchDepositContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BatchDepositContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BatchDepositContractSession struct {
	Contract     *BatchDepositContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// BatchDepositContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BatchDepositContractCallerSession struct {
	Contract *BatchDepositContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// BatchDepositContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BatchDepositContractTransactorSession struct {
	Contract     *BatchDepositContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// BatchDepositContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type BatchDepositContractRaw struct {
	Contract *BatchDepositContract // Generic contract binding to access the raw methods on
}

// BatchDepositContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BatchDepositContractCallerRaw struct {
	Contract *BatchDepositContractCaller // Generic read-only contract binding to access the raw methods on
}

// BatchDepositContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BatchDepositContractTransactorRaw struct {
	Contract *BatchDepositContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBatchDepositContract creates a new instance of BatchDepositContract, bound to a specific deployed contract.
func NewBatchDepositContract(address common.Address, backend bind.ContractBackend) (*BatchDepositContract, error) {
	contract, err := bindBatchDepositContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BatchDepositContract{BatchDepositContractCaller: BatchDepositContractCaller{contract: contract}, BatchDepositContractTransactor: BatchDepositContractTransactor{contract: contract}, BatchDepositContractFilterer: BatchDepositContractFilterer{contract: contract}}, nil
}

// NewBatchDepositContractCaller creates a new read-only instance of BatchDepositContract, bound to a specific deployed contract.
func NewBatchDepositContractCaller(address common.Address, caller bind.ContractCaller) (*BatchDepositContractCaller, error) {
	contract, err := bindBatchDepositContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BatchDepositContractCaller{contract: contract}, nil
}

// NewBatchDepositContractTransactor creates a new write-only instance of BatchDepositContract, bound to a specific deployed contract.
func NewBatchDepositContractTransactor(address common.Address, transactor bind.ContractTransactor) (*BatchDepositContractTransactor, error) {
	contract, err := bindBatchDepositContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BatchDepositContractTransactor{contract: contract}, nil
}

// NewBatchDepositContractFilterer creates a new log filterer instance of BatchDepositContract, bound to a specific deployed contract.
func NewBatchDepositContractFilterer(address common.Address, filterer bind.ContractFilterer) (*BatchDepositContractFilterer, error) {
	contract, err := bindBatchDepositContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BatchDepositContractFilterer{contract: contract}, nil
}

// bindBatchDepositContract binds a generic wrapper to an already deployed contract.
func bindBatchDepositContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BatchDepositContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BatchDepositContract *BatchDepositContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BatchDepositContract.Contract.BatchDepositContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BatchDepositContract *BatchDepositContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BatchDepositContract.Contract.BatchDepositContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BatchDepositContract *BatchDepositContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BatchDepositContract.Contract.BatchDepositContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BatchDepositContract *BatchDepositContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BatchDepositContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BatchDepositContract *BatchDepositContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BatchDepositContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BatchDepositContract *BatchDepositContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BatchDepositContract.Contract.contract.Transact(opts, method, params...)
}

// DepositContract is a free data retrieval call binding the contract method 0xe94ad65b.
//
// Solidity: function depositContract() view returns(address)
func (_BatchDepositContract *BatchDepositContractCaller) DepositContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BatchDepositContract.contract.Call(opts, &out, "depositContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DepositContract is a free data retrieval call binding the contract method 0xe94ad65b.
//
// Solidity: function depositContract() view returns(address)
func (_BatchDepositContract *BatchDepositContractSession) DepositContract() (common.Address, error) {
	return _BatchDepositContract.Contract.DepositContract(&_BatchDepositContract.CallOpts)
}

// DepositContract is a free data retrieval call binding the contract method 0xe94ad65b.
//
// Solidity: function depositContract() view returns(address)
func (_BatchDepositContract *BatchDepositContractCallerSession) DepositContract() (common.Address, error) {
	return _BatchDepositContract.Contract.DepositContract(&_BatchDepositContract.CallOpts)
}

// BatchDeposit is a paid mutator transaction binding the contract method 0xddbd9dd9.
//
// Solidity: function batchDeposit(bytes pubkeys, bytes signatures, bytes32[] dataRoots, bytes withdrawalCredentials, uint256 amountWei) payable returns()
func (_BatchDepositContract *BatchDepositContractTransactor) BatchDeposit(opts *bind.TransactOpts, pubkeys []byte, signatures []byte, dataRoots [][32]byte, withdrawalCredentials []byte, amountWei *big.Int) (*types.Transaction, error) {
	return _BatchDepositContract.contract.Transact(opts, "batchDeposit", pubkeys, signatures, dataRoots, withdrawalCredentials, amountWei)
}

// BatchDeposit is a paid mutator transaction binding the contract method 0xddbd9dd9.
//
// Solidity: function batchDeposit(bytes pubkeys, bytes signatures, bytes32[] dataRoots, bytes withdrawalCredentials, uint256 amountWei) payable returns()
func (_BatchDepositContract *BatchDepositContractSession) BatchDeposit(pubkeys []byte, signatures []byte, dataRoots [][32]byte, withdrawalCredentials []byte, amountWei *big.Int) (*types.Transaction, error) {
	return _BatchDepositContract.Contract.BatchDeposit(&_BatchDepositContract.TransactOpts, pubkeys, signatures, dataRoots, withdrawalCredentials, amountWei)
}

// BatchDeposit is a paid mutator transaction binding the contract method 0xddbd9dd9.
//
// Solidity: function batchDeposit(bytes pubkeys, bytes signatures, bytes32[] dataRoots, bytes withdrawalCredentials, uint256 amountWei) payable returns()
func (_BatchDepositContract *BatchDepositContractTransactorSession) BatchDeposit(pubkeys []byte, signatures []byte, dataRoots [][32]byte, withdrawalCredentials []byte, amountWei *big.Int) (*types.Transaction, error) {
	return _BatchDepositContract.Contract.BatchDeposit(&_BatchDepositContract.TransactOpts, pubkeys, signatures, dataRoots, withdrawalCredentials, amountWei)
}
