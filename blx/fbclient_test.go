package blx

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"strings"
	"testing"
)

func TestGetBalance(t *testing.T) {

	address := "0xFd2955B33Fa3bE18b6ef3a90097F8a25F5E5FF85"
	jk := NewJk(2, MainNet)
	of, err := jk.GetBalanceOf(context.Background(), address)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestGetBalanceOfContract(t *testing.T) {

	address := "0x116663f85a8727410efa33f7051265efae77ed98"
	contractAddr := "0x03007fcaa04cec04820ed54e1a49b2e0f69cc298"
	jk := NewJk(2, MainNet)
	of, _, err := jk.GetBalanceOfContract(context.Background(), address, contractAddr)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestJk_GetSymbolOfContract(t *testing.T) {
	contractAddr := "0x03007fcaa04cec04820ed54e1a49b2e0f69cc298"
	jk := NewJk(2, MainNet)
	of, err := jk.GetSymbolOfContract(context.Background(), contractAddr)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestSendContractInputDataSync(t *testing.T) {
	//inputData := "0x80cdddf8000000000000000000000000af6ec332596f3a46ff9d36d8592d1ba5765473ac0000000000000000000000008394a30ea38c23164d178651fb9c6c826d8096960000000000000000000000000000000000000000000000000000000005f5e100"
	bs := []byte(`[
	{
		"inputs": [
			{
				"internalType": "contract IERC20",
				"name": "_lockContract",
				"type": "address"
			},
			{
				"internalType": "address",
				"name": "_mappedAddress",
				"type": "address"
			}
		],
		"stateMutability": "nonpayable",
		"type": "constructor"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "address",
				"name": "previousOwner",
				"type": "address"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "newOwner",
				"type": "address"
			}
		],
		"name": "OwnershipTransferred",
		"type": "event"
	},
	{
		"inputs": [
			{
				"internalType": "contract IERC20",
				"name": "contractAddress",
				"type": "address"
			},
			{
				"internalType": "address",
				"name": "sender",
				"type": "address"
			}
		],
		"name": "ApproveFrc20",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"internalType": "string",
				"name": "id",
				"type": "string"
			}
		],
		"name": "Deposit",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "contract IERC20",
				"name": "contractAddress",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"internalType": "string",
				"name": "id",
				"type": "string"
			}
		],
		"name": "DepositFrc20",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "query",
				"type": "address"
			}
		],
		"name": "GetAddressBalance",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "contract IERC20",
				"name": "contractAddress",
				"type": "address"
			},
			{
				"internalType": "address",
				"name": "query",
				"type": "address"
			}
		],
		"name": "GetAddressFrc20Balance",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "GetBalance",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "contract IERC20",
				"name": "contractAddress",
				"type": "address"
			}
		],
		"name": "GetFrc20Balance",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "GetLockAddress",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "query",
				"type": "address"
			}
		],
		"name": "GetLockBalance",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "GetMappedAddress",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "query",
				"type": "address"
			}
		],
		"name": "GetOracleAction",
		"outputs": [
			{
				"internalType": "string",
				"name": "",
				"type": "string"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"internalType": "string",
				"name": "year",
				"type": "string"
			}
		],
		"name": "Lock",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"internalType": "string",
				"name": "id",
				"type": "string"
			}
		],
		"name": "Mapped",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "contract IERC20",
				"name": "contractAddress",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"internalType": "string",
				"name": "id",
				"type": "string"
			}
		],
		"name": "MappedFrc20",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "action",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "id",
				"type": "string"
			}
		],
		"name": "OracleAction",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "sender",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "year",
				"type": "uint256"
			}
		],
		"name": "UnLock",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "UnsafeOpen",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address payable",
				"name": "sender",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			}
		],
		"name": "UnsafeWithdraw",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "UnsafeWithdrawAll",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "contract IERC20",
				"name": "contractAddress",
				"type": "address"
			},
			{
				"internalType": "address",
				"name": "sender",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			}
		],
		"name": "UnsafeWithdrawFrc20",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "contract IERC20",
				"name": "contractAddress",
				"type": "address"
			},
			{
				"internalType": "address",
				"name": "sender",
				"type": "address"
			}
		],
		"name": "UnsafeWithdrawFrc20All",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address payable",
				"name": "sender",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			}
		],
		"name": "Withdraw",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "contract IERC20",
				"name": "contractAddress",
				"type": "address"
			},
			{
				"internalType": "address",
				"name": "sender",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			}
		],
		"name": "WithdrawFrc20",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "owner",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "renounceOwnership",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "newOwner",
				"type": "address"
			}
		],
		"name": "transferOwnership",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`)

	abiJsons, err := abi.JSON(bytes.NewBuffer(bs))
	if err != nil {
		t.Error(err)
	}

	amount, _ := big.NewInt(0).SetString("100000000", 10)
	inputData, err := abiJsons.Pack("WithdrawFrc20", common.HexToAddress("0xaf6ec332596f3a46ff9d36d8592d1ba5765473ac"), common.HexToAddress("0x8394a30Ea38c23164d178651FB9c6c826d809696"), amount)
	if err != nil {
		t.Error(err)
	}

	//encodeHex := hex.EncodeToString(inputData)
	//t.Log("inputData", "0x" + encodeHex)
	//
	//unpack, err := abiJsons.Unpack("WithdrawFrc20", inputData)
	//if err != nil {
	//	t.Error(err)
	//}
	//t.Log("unpack", unpack)

	senderPrivate := "9515496153707b43962be4426e247f0a8ce3ce0968fcf53531d524c1df276d0c"
	contractAddr := "0xb194A5113C373494ceE66B516a4e8c203b1182b1"

	jk := NewJk(2, MainNet)
	hash, err := jk.SendContractInputDataSync(context.Background(), senderPrivate, inputData, contractAddr)
	if err != nil {
		t.Error(err)
	}

	t.Log(hash)
}

func TestJk_SendSync(t *testing.T) {

	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"
	jk := NewJk(2, MainNet)
	of, err := jk.SendSync(context.Background(), senderPrivate, to, 1.1)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestJk_SendAsync(t *testing.T) {

	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"

	jk := NewJk(2, MainNet)
	of, err := jk.SendAsync(context.Background(), senderPrivate, to, 1.1, 0)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestSendContractSync(t *testing.T) {
	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"
	contractAddr := "0x03007fcaa04cec04820ed54e1a49b2e0f69cc298"

	jk := NewJk(2, MainNet)
	of, err := jk.SendContractSync(context.Background(), senderPrivate, to, 1.1, contractAddr)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestSendContractAsync(t *testing.T) {
	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"
	contractAddr := "0x03007fcaa04cec04820ed54e1a49b2e0f69cc298"

	jk := NewJk(2, MainNet)
	of, err := jk.SendContractAsync(context.Background(), senderPrivate, to, 1.1, 0, contractAddr)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestJk_GetTransactionByHash(t *testing.T) {
	//hash := "0xaf7a4ec9108db2283bc606f61e439357b796ee1598fc6fd6839e3e78121f819c"

	hash := "0xb0f48964f300df86fcfee6d3f0cfd57acdf53f8474f8787342ac6d8d6a0fede7"

	jk := NewJk(2, MainNet)
	tx, pending, err := jk.GetTransactionByHash(context.Background(), hash)
	if err != nil {
		t.Error(err)
	}

	t.Log("pending: \n", pending)

	fmt.Printf("tx chainId: %v \n", tx.ChainId())
	fmt.Printf("tx gas price: %v \n", tx.GasPrice())
	fmt.Printf("tx gas: %v \n", tx.Gas())
	fmt.Printf("tx to: %v \n", tx.To())

	//0x02411f5a0000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000013100000000000000000000000000000000000000000000000000000000000000
	fmt.Printf("tx data: %v \n", hex.EncodeToString(tx.Data()))

	fmt.Printf("tx value: %v \n", tx.Value())
	fmt.Printf("tx cost: %v \n", tx.Cost())
}

func TestJk_GetTransactionByReceipt(t *testing.T) {
	hash := "0xe5ecd42ebcfa7d8d41bf1580425d05c79fe1ccbee61d25dbfee584978f344ec3"

	//hash := "0xda34db2a63ae63010388b89633dd35826a86a0107a031bc0e6a53d80a7279f0f"

	jk := NewJk(2, MainNet)
	tx, err := jk.GetTransactionReceiptByHash(context.Background(), hash)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("tx: %+v \n", tx)
}

func TestJk_GetBlockByHash(t *testing.T) {
	hash := "0x86fac4cf8bab415d2c92ab81e715c295230d294003c616833a3dc225007b3c8c"

	jk := NewJk(2, MainNet)
	block, err := jk.GetBlockByHash(context.Background(), hash)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%+v height: %v \n", block, block.Number())

	transactions := block.Transactions()
	for _, tx := range transactions {

		data := tx.Data()
		fmt.Printf("tx hash: %v \n", tx.Hash())
		fmt.Printf("tx data: %v \n", data)

		//try parse erc20 data
		erc20Abi, err := abi.JSON(strings.NewReader(AbiErc20))
		if err != nil {
			t.Error("json abi err: ", err)
		}

		method, ok := erc20Abi.Methods["transfer"]
		if ok {
			params, err := method.Inputs.Unpack(data[4:])
			if err != nil {
				t.Error(err)
			}

			t.Log(params)
		}

		fmt.Printf("tx chainId: %v \n", tx.ChainId())
		fmt.Printf("tx gas price: %v \n", tx.GasPrice())
		fmt.Printf("tx gas: %v \n", tx.Gas())

		fmt.Printf("tx value: %v \n", tx.Value())
		fmt.Printf("tx cost: %v \n", tx.Cost())

	}
}

func TestSendContractSyncBatch(t *testing.T) {
	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"
	contractAddr := "0x03007fcaa04cec04820ed54e1a49b2e0f69cc298"

	jk := NewJk(2, MainNet)

	for i := 0; i < 10; i++ {
		of, err := jk.SendContractSync(context.Background(), senderPrivate, to, 1.1, contractAddr)
		if err != nil {
			t.Error(err)
		}

		t.Log(of)
	}

}

func TestSendContractAsyncBatch(t *testing.T) {

	jk := NewJk(2, MainNet)

	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"
	contractAddr := "0x03007fcaa04cec04820ed54e1a49b2e0f69cc298"

	privateKey, err := crypto.HexToECDSA(senderPrivate)
	if err != nil {
		t.Error(err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Error(err)
	}

	sender := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	nonce, err := jk.GetPendingNonce(context.Background(), sender)
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 10; i++ {

		of, err := jk.SendContractAsync(context.Background(), senderPrivate, to, 1, nonce, contractAddr)
		if err != nil {
			t.Error(err)
		}

		nonce++

		t.Log(of)
	}

}

func TestJk_SendSyncBalanceNotEnough(t *testing.T) {

	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"

	jk := NewJk(2, MainNet)
	of, err := jk.SendSync(context.Background(), senderPrivate, to, 50)
	if err != nil {
		if err == BalanceLessGasAddAmountError {

		} else {
			t.Error(err)
		}
	}

	t.Log(of)
}

func TestStartScan(t *testing.T) {
	jk := NewJk(2, MainNet)
	err := jk.StartScan(1, func(tx *types.Transaction, block *types.Block) error {

		msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId()), block.BaseFee())
		if err != nil {
			t.Error("decode to message: ", err)
			return err
		}

		height := block.Number()
		hash := tx.Hash()
		value := tx.Value()
		to := tx.To()
		nonce := tx.Nonce()
		data := tx.Data()
		from := msg.From()

		t.Log(fmt.Sprintf("from: %v hash: %v value: %v  to: %v nonce: %v  data: %v height: %v", from, hash, value, to, nonce, data, height))

		return nil
	})

	if err != nil {
		t.Error(err)
	}
}

func TestStartScanError(t *testing.T) {
	jk := NewJk(2, MainNet)
	err := jk.StartScan(330600, func(tx *types.Transaction, block *types.Block) error {

		msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId()), block.BaseFee())
		if err != nil {
			t.Error("decode to message: ", err)
			return err
		}

		height := block.Number()
		hash := tx.Hash()
		value := tx.Value()
		to := tx.To()
		nonce := tx.Nonce()
		data := tx.Data()
		from := msg.From()

		t.Log(fmt.Sprintf("from: %v hash: %v value: %v  to: %v nonce: %v  data: %v height: %v", from, hash, value, to, nonce, data, height))

		return errors.New(fmt.Sprintf("from: %v hash: %v value: %v  to: %v nonce: %v  data: %v height: %v", from, hash, value, to, nonce, data, height))
	})

	if err != nil {
		t.Error(err)
	}
}

func TestJk_IsContract(t *testing.T) {
	jk := NewJk(2, MainNet)

	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"
	contractAddr := "0x03007fcaa04cec04820ed54e1a49b2e0f69cc298"

	isC, err := jk.IsContract(context.Background(), to)
	if err != nil {
		t.Error(err)
	}

	if isC {
		t.Error(to, "not a contract address")
	}

	isCC, err := jk.IsContract(context.Background(), contractAddr)
	if err != nil {
		t.Error(err)
	}

	if !isCC {
		t.Error(contractAddr, "is a contract address")
	}

}
