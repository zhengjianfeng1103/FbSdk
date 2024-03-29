package blx

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
	"github.com/zhengjianfeng1103/FbSdk/log"
)

const MainNet = "https://node.fibochain.org"
const TestNet = "https://test.fibochain.org"
const MainNet2 = "http://13.228.22.173:8545"

const MainNetCoin = "FIBO"
const MainNetChainId = 12306
const MaxRetrySync = 24
const MaxRetryTimeDurationSeconds = 1

var MainCoinDecimal = big.NewFloat(math.Pow(10, 18))

var PoolClosedError = NewJkError("请求池子用尽")
var BalanceLessGasError = NewJkError("交易费不足")
var BalanceLessGasAddAmountError = NewJkError("余额小于交易费+转账数量")
var BalanceLessAmountError = NewJkError("转账数量不足")
var PrivateKeyError = NewJkError("错误的私钥")
var AmountError = NewJkError("金额有误")
var ReadTransactionTimeOutError = NewJkError("读取交易信息失败")
var SendTransactionFailedError = NewJkError("交易失败")
var ContractNotEmpty = NewJkError("合约地址不能为空")
var NotAnHexAddress = NewJkError("不是合法0x地址")
var NonceNotEmpty = NewJkError("交易序号不能为空")
var NonceToSmall = NewJkError("交易序号太小")

type Jk struct {
	cons    chan *ethclient.Client
	factory func() (*ethclient.Client, error)
	m       sync.Mutex
	rw      sync.Mutex
	closed  bool
}

func NewJk(size int, net string, level logrus.Level) *Jk {
	log.Init(level)
	cons := make(chan *ethclient.Client, size)

	if size == 0 {
		size = 3
	}

	for i := 0; i < size; i++ {

		timeoutC, fn := context.WithTimeout(context.Background(), 10*time.Second)
		fn()
		connect, err := rpc.DialContext(timeoutC, net)
		if err != nil {
			log.Log.Error("connect eth mainNet error", err)
			continue
		}

		ec := ethclient.NewClient(connect)
		if err != nil {
			log.Log.Debug(fmt.Sprintf("init eth client err: %v", err))
		}
		cons <- ec
	}

	return &Jk{cons, func() (*ethclient.Client, error) {

		timeoutC, fn := context.WithTimeout(context.Background(), 10*time.Second)
		fn()
		connect, err := rpc.DialContext(timeoutC, MainNet)
		if err != nil {
			log.Log.Error("connect eth mainNet error", err)
			return nil, err
		}

		ec := ethclient.NewClient(connect)
		if err != nil {
			log.Log.Debug(fmt.Sprintf("init eth client err: %v", err))
			return nil, err
		}
		return ec, nil
	},
		sync.Mutex{},
		sync.Mutex{},
		false}
}

func (j *Jk) Acquire() (*ethclient.Client, error) {
	select {
	case r, ok := <-j.cons:
		if !ok {
			return nil, PoolClosedError
		}
		log.Log.Debug("从池子里获取连接了")
		return r, nil
	default:
		log.Log.Debug("需要新建资源了")
		return j.factory()
	}
}

func (j *Jk) Release(r *ethclient.Client) {
	//保证该操作和Close方法的操作是安全的
	j.m.Lock()
	defer j.m.Unlock()

	//资源池都关闭了，就省这一个没有释放的资源了，释放即可
	if j.closed {
		r.Close()
		return
	}

	select {
	case j.cons <- r:
		log.Log.Debug("资源释放到池子里了")
	default:
		log.Log.Debug("资源池满了，释放这个资源吧")
		r.Close()
	}
}

func (j *Jk) Close() {
	j.m.Lock()
	defer j.m.Unlock()

	if j.closed {
		return
	}

	j.closed = true

	//关闭通道，不让写入了
	close(j.cons)

	//关闭通道里的资源
	for r := range j.cons {
		r.Close()
	}
}

func (j *Jk) GetBalanceOf(ctx context.Context, address string) (balance float64, err error) {
	client, err := j.Acquire()
	if err != nil {
		return 0, err
	}
	defer j.Release(client)

	bc, err := client.BalanceAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return
	}

	flb, success := new(big.Float).SetString(bc.String())
	if !success {
		return 0, errors.New("balance can not covert to big float")
	}

	bcf, _ := new(big.Float).Quo(flb, MainCoinDecimal).Float64()
	return bcf, nil
}

func (j *Jk) GetPendingNonce(ctx context.Context, address string) (nonce uint64, err error) {
	client, err := j.Acquire()
	if err != nil {
		return 0, err
	}
	defer j.Release(client)

	at, err := client.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		return 0, err
	}
	return at, nil
}

func (j *Jk) GetBalanceOfContract(ctx context.Context, address string, contractAddr string) (balance float64, decimals float64, err error) {
	client, err := j.Acquire()
	if err != nil {
		return
	}
	defer j.Release(client)

	if contractAddr == "" {
		return 0, 0, ContractNotEmpty
	}

	from := common.HexToAddress(address)
	to := common.HexToAddress(contractAddr)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	log.Log.Debug("gasPrice", gasPrice)

	erc20Abi, err := abi.JSON(strings.NewReader(AbiErc20))
	if err != nil {
		return
	}

	input, err := erc20Abi.Pack("balanceOf", from)
	if err != nil {
		return
	}

	msg := ethereum.CallMsg{
		From:     from,
		To:       &to,
		Data:     input,
		GasPrice: gasPrice,
		Gas:      10000,
	}

	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return
	}

	//call for balanceOf
	unpack, err := erc20Abi.Unpack("balanceOf", result)
	if err != nil {
		return
	}

	//call for decimals
	inputDecimals, err := erc20Abi.Pack("decimals")
	if err != nil {
		return
	}
	msgDecimals := ethereum.CallMsg{
		From: from,
		To:   &to,
		Data: inputDecimals,
	}

	resultDecimals, err := client.CallContract(ctx, msgDecimals, nil)
	if err != nil {
		return
	}

	//call for balanceOf
	unpackDecimals, err := erc20Abi.Unpack("decimals", resultDecimals)
	if err != nil {
		return
	}

	log.Log.Debug("call decimals of result: ", unpackDecimals[0])
	log.Log.Debug("call balance of result: ", unpack[0])

	bc := unpack[0].(*big.Int)
	fc := new(big.Float).SetInt(bc)

	bDecimals := unpackDecimals[0].(uint8)
	fDecimals := new(big.Float).SetFloat64(math.Pow10(int(bDecimals)))
	fDecimalsF, _ := fDecimals.Float64()

	f, _ := new(big.Float).Quo(fc, fDecimals).Float64()
	return f, fDecimalsF, nil
}

func (j *Jk) GetSymbolOfContract(ctx context.Context, contractAddr string) (symbol string, err error) {
	client, err := j.Acquire()
	if err != nil {
		return
	}
	defer j.Release(client)

	if contractAddr == "" {
		return "", ContractNotEmpty
	}

	to := common.HexToAddress(contractAddr)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	log.Log.Debug("gasPrice", gasPrice)

	erc20Abi, err := abi.JSON(strings.NewReader(AbiErc20))
	if err != nil {
		return
	}

	input, err := erc20Abi.Pack("symbol")
	if err != nil {
		return
	}

	msg := ethereum.CallMsg{
		To:   &to,
		Data: input,
	}

	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return
	}

	//call for symbol
	unpack, err := erc20Abi.Unpack("symbol", result)
	if err != nil {
		return
	}

	log.Log.Debug("call symbol of result: ", unpack[0])

	bc := unpack[0].(string)
	return bc, nil
}

func (j *Jk) SendSync(ctx context.Context, senderPrivate string, receive string, amount float64) (hash string, err error) {
	if !common.IsHexAddress(receive) {
		return "", NotAnHexAddress
	}
	client, err := j.Acquire()
	if err != nil {
		return
	}
	defer j.Release(client)

	if strings.HasPrefix(senderPrivate, "0x") {
		senderPrivate = senderPrivate[2:]
	}

	privateKey, err := crypto.HexToECDSA(senderPrivate)
	if err != nil {
		log.Log.Error(fmt.Sprintf("recover key err: %v", err))
		return "", PrivateKeyError
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Log.Error("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return "", PrivateKeyError
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	log.Log.Debug("Public Key: ", hexutil.Encode(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	log.Log.Debug("Address: ", address)

	from := common.HexToAddress(address)
	to := common.HexToAddress(receive)

	strAmount := fmt.Sprintf("%f", amount)
	f, success := new(big.Float).SetString(strAmount)
	if !success {
		return "", AmountError
	}
	fCoins := new(big.Float).Mul(f, MainCoinDecimal)
	coins, _ := fCoins.Int(nil)

	log.Log.Debug("from: ", from, "to: ", to, "coins: ", coins)

	gasPrice, err := client.SuggestGasPrice(ctx)
	log.Log.Debug("gasPrice: ", gasPrice)

	balance, err := client.BalanceAt(ctx, from, nil)
	log.Log.Debug("balance: ", balance)

	pendingNonce, err := client.PendingNonceAt(ctx, from)
	log.Log.Debug("pendingNonce: ", pendingNonce)

	gasLimit := uint64(30000)

	gas := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
	if balance.Cmp(gas) <= 0 {
		return "", BalanceLessGasError
	}

	log.Log.Debug("gas: ", gas)

	if balance.Cmp(new(big.Int).Add(gas, coins)) <= 0 {
		return "", BalanceLessGasAddAmountError
	}

	unsignedTx := types.NewTransaction(pendingNonce, to, coins, gasLimit, gasPrice, []byte{})
	signedTx, _ := types.SignTx(unsignedTx, types.NewEIP155Signer(big.NewInt(MainNetChainId)), privateKey)

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Log.Error("send transaction", err)
		return "", err
	}

	txHash := signedTx.Hash()
	log.Log.Debug("sendTx txHash:", txHash)

	hash = txHash.Hex()
	retryTimes := 0
	for {
		select {
		case <-time.NewTimer(MaxRetryTimeDurationSeconds * time.Second).C:
			if retryTimes >= MaxRetrySync {
				return hash, nil
			}

			receipt, err := client.TransactionReceipt(ctx, txHash)
			if err != nil {
				log.Log.Error("get transaction receipt: ", err)
				retryTimes++
				continue
			}

			if receipt.Status == types.ReceiptStatusSuccessful && receipt.BlockNumber != nil {
				log.Log.Debug("get transaction success status", " Tx In BlockNumber: ", receipt.BlockNumber, " GasUse: ", receipt.GasUsed, " Logs: ", receipt.Logs)
				return hash, nil
			}

			if receipt.Status == types.ReceiptStatusFailed {
				log.Log.Error("get transaction failed status")
				return "", SendTransactionFailedError
			}

		case <-ctx.Done():
			log.Log.Error("get transaction time out context")
			return "", ReadTransactionTimeOutError
		}
	}
}

// SendRawTx rawTx hexString no 0x
func (j *Jk) SendRawTx(ctx context.Context, rawTx string) (hash string, txResult *types.Transaction, err error) {
	client, err := j.Acquire()
	if err != nil {
		return
	}
	defer j.Release(client)

	var tx *types.Transaction
	rawTxBytes, err := hex.DecodeString(rawTx)
	if err != nil {
		return "", nil, err
	}
	err = rlp.DecodeBytes(rawTxBytes, &tx)
	if err != nil {
		return "", nil, err
	}

	hash = tx.Hash().String()
	log.Log.Debug("sendRawTx: ", tx.Hash().String())

	err = client.SendTransaction(context.Background(), tx)
	if err != nil {
		return hash, tx, err
	}

	retryTimes := 0
	for {
		select {
		case <-time.NewTimer(MaxRetryTimeDurationSeconds * time.Second).C:
			if retryTimes >= MaxRetrySync {
				return hash, tx, nil
			}

			receipt, err := j.GetTransactionReceiptByHash(ctx, hash)
			if err != nil {
				log.Log.Error("get transaction receipt: ", err)
				retryTimes++
				continue
			}

			if receipt.Status == types.ReceiptStatusSuccessful && receipt.BlockNumber != nil {
				log.Log.Debug("get transaction success status", " Tx In BlockNumber: ", receipt.BlockNumber, " GasUse: ", receipt.GasUsed, " Logs: ", receipt.Logs)
				return hash, tx, nil
			}

			if receipt.Status == types.ReceiptStatusFailed {
				log.Log.Error("get transaction failed status")
				return hash, tx, SendTransactionFailedError
			}

		case <-ctx.Done():
			log.Log.Error("get transaction time out context")
			return hash, tx, ReadTransactionTimeOutError
		}
	}
}

func (j *Jk) SendAsync(ctx context.Context, senderPrivate string, receive string, amount float64, nonce uint64) (hash string, err error) {
	if !common.IsHexAddress(receive) {
		return "", NotAnHexAddress
	}

	client, err := j.Acquire()
	if err != nil {
		return
	}
	defer j.Release(client)

	if strings.HasPrefix(senderPrivate, "0x") {
		senderPrivate = senderPrivate[2:]
	}
	privateKey, err := crypto.HexToECDSA(senderPrivate)
	if err != nil {
		log.Log.Error(fmt.Sprintf("recover key err: %v", err))
		return "", PrivateKeyError
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Log.Error("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return "", PrivateKeyError
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	log.Log.Debug("Public Key: ", hexutil.Encode(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	log.Log.Debug("Address: ", address)
	from := common.HexToAddress(address)
	to := common.HexToAddress(receive)

	strAmount := fmt.Sprintf("%f", amount)
	f, success := new(big.Float).SetString(strAmount)
	if !success {
		return "", AmountError
	}
	fCoins := new(big.Float).Mul(f, MainCoinDecimal)
	coins, _ := fCoins.Int(nil)

	log.Log.Debug("from: ", from, "to: ", to, "coins: ", coins)

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Log.Error("get balance err: ", err)
		return "", err
	}
	log.Log.Debug("gasPrice: ", gasPrice)

	balance, err := client.BalanceAt(ctx, from, nil)
	if err != nil {
		log.Log.Error("get balance err: ", err)
		return "", err
	}
	log.Log.Debug("balance: ", balance)

	var pendingNonce uint64
	if nonce == 0 {
		pendingNonce, err = client.PendingNonceAt(ctx, from)
		if err != nil {
			log.Log.Error("get pendingNonce err: ", err)
			return "", err
		}
	} else {
		pendingNonce = nonce
	}
	log.Log.Debug("pendingNonce: ", pendingNonce)

	gasLimit := uint64(30000)
	gas := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
	if balance.Cmp(gas) <= 0 {
		return "", BalanceLessGasError
	}

	log.Log.Debug("gas: ", gas)

	if balance.Cmp(new(big.Int).Add(gas, coins)) <= 0 {
		return "", BalanceLessGasAddAmountError
	}

	unsignedTx := types.NewTransaction(pendingNonce, to, coins, gasLimit, gasPrice, []byte{})
	signedTx, _ := types.SignTx(unsignedTx, types.NewEIP155Signer(big.NewInt(MainNetChainId)), privateKey)

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Log.Error("send transaction", err)
		return "", err
	}

	txHash := signedTx.Hash()
	log.Log.Debug("sendTx txHash:", txHash)

	hash = txHash.Hex()
	return
}

func (j *Jk) SendContractSync(ctx context.Context, senderPrivate string, receive string, amount float64, contractAddr string) (hash string, err error) {
	if !common.IsHexAddress(receive) {
		return "", NotAnHexAddress
	}

	if contractAddr == "" {
		return "", ContractNotEmpty
	}

	client, err := j.Acquire()
	if err != nil {
		return
	}
	defer j.Release(client)

	if strings.HasPrefix(senderPrivate, "0x") {
		senderPrivate = senderPrivate[2:]
	}

	privateKey, err := crypto.HexToECDSA(senderPrivate)
	if err != nil {
		log.Log.Error(fmt.Sprintf("recover key err: %v", err))
		return "", PrivateKeyError
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Log.Error("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return "", PrivateKeyError
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	log.Log.Debug("Public Key: ", hexutil.Encode(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	log.Log.Debug("Address: ", address)

	balanceContract, decimals, err := j.GetBalanceOfContract(ctx, address, contractAddr)
	if err != nil {
		return "", err
	}

	if balanceContract < amount {
		return "", BalanceLessAmountError
	}

	from := common.HexToAddress(address)
	to := common.HexToAddress(receive)
	contract := common.HexToAddress(contractAddr)

	strAmount := fmt.Sprintf("%f", amount)
	f, success := new(big.Float).SetString(strAmount)
	if !success {
		return "", AmountError
	}

	fCoins := new(big.Float).Mul(f, big.NewFloat(decimals))
	coins, _ := fCoins.Int(nil)

	log.Log.Debug("from: ", from, " to: ", to, " contractAddr: ", contractAddr, " coins: ", coins)

	gasPrice, err := client.SuggestGasPrice(ctx)
	log.Log.Debug("gasPrice: ", gasPrice)

	balance, err := client.BalanceAt(ctx, from, nil)
	log.Log.Debug("balance: ", balance)

	pendingNonce, err := client.PendingNonceAt(ctx, from)
	log.Log.Debug("pendingNonce: ", pendingNonce)

	erc20Abi, err := abi.JSON(strings.NewReader(AbiErc20))
	if err != nil {
		return
	}

	input, err := erc20Abi.Pack("transfer", to, coins)
	if err != nil {
		return
	}

	msgGas := ethereum.CallMsg{
		From: from,
		To:   &contract,
		Data: input,
	}

	gasLimit, err := client.EstimateGas(ctx, msgGas)
	log.Log.Debug("gasLimit: ", gasLimit)

	if err != nil {
		return "", err
	}

	gas := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
	log.Log.Debug("gas: ", gas)

	if balance.Cmp(gas) <= 0 {
		return "", BalanceLessGasError
	}

	unsignedTx := types.NewTransaction(pendingNonce, contract, big.NewInt(0), gasLimit, gasPrice, input)
	signedTx, _ := types.SignTx(unsignedTx, types.NewEIP155Signer(big.NewInt(MainNetChainId)), privateKey)

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Log.Error("send transaction", err)
		return "", err
	}

	txHash := signedTx.Hash()
	log.Log.Debug("sendTx txHash:", txHash)

	hash = txHash.Hex()
	retryTimes := 0
	for {
		select {
		case <-time.NewTimer(MaxRetryTimeDurationSeconds * time.Second).C:
			if retryTimes >= MaxRetrySync {
				return hash, nil
			}

			receipt, err := client.TransactionReceipt(ctx, txHash)
			if err != nil {
				log.Log.Error("get transaction receipt: ", err)
				retryTimes++
				continue
			}

			if receipt.Status == types.ReceiptStatusSuccessful && receipt.BlockNumber != nil {
				log.Log.Debug("get transaction success status", " Tx In BlockNumber: ", receipt.BlockNumber, " GasUse: ", receipt.GasUsed, " Logs: ", receipt.Logs)
				return hash, nil
			}

			if receipt.Status == types.ReceiptStatusFailed {
				log.Log.Error("get transaction failed status")
				return "", SendTransactionFailedError
			}

		case <-ctx.Done():
			log.Log.Error("get transaction time out context")
			return "", ReadTransactionTimeOutError
		}
	}
}

func (j *Jk) SendContractSyncWithNonce(ctx context.Context, senderPrivate string, receive string, amount float64, contractAddr string, pendingNonce uint64) (hash string, err error) {
	if !common.IsHexAddress(receive) {
		return "", NotAnHexAddress
	}

	if contractAddr == "" {
		return "", ContractNotEmpty
	}

	if pendingNonce == 0 {
		return "", NonceNotEmpty
	}

	client, err := j.Acquire()
	if err != nil {
		return
	}
	defer j.Release(client)

	if strings.HasPrefix(senderPrivate, "0x") {
		senderPrivate = senderPrivate[2:]
	}

	privateKey, err := crypto.HexToECDSA(senderPrivate)
	if err != nil {
		log.Log.Error(fmt.Sprintf("recover key err: %v", err))
		return "", PrivateKeyError
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Log.Error("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return "", PrivateKeyError
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	log.Log.Debug("Public Key: ", hexutil.Encode(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	log.Log.Debug("Address: ", address)

	balanceContract, decimals, err := j.GetBalanceOfContract(ctx, address, contractAddr)
	if err != nil {
		return "", err
	}

	if balanceContract < amount {
		return "", BalanceLessAmountError
	}

	from := common.HexToAddress(address)
	to := common.HexToAddress(receive)
	contract := common.HexToAddress(contractAddr)

	strAmount := fmt.Sprintf("%f", amount)
	f, success := new(big.Float).SetString(strAmount)
	if !success {
		return "", AmountError
	}

	fCoins := new(big.Float).Mul(f, big.NewFloat(decimals))
	coins, _ := fCoins.Int(nil)

	log.Log.Debug("from: ", from, " to: ", to, " contractAddr: ", contractAddr, " coins: ", coins)

	gasPrice, err := client.SuggestGasPrice(ctx)
	log.Log.Debug("gasPrice: ", gasPrice)

	balance, err := client.BalanceAt(ctx, from, nil)
	log.Log.Debug("balance: ", balance)

	pendingNonceNew, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", err
	}

	log.Log.Debug("pendingNonce: ", pendingNonce, " pendingNonceNew: ", pendingNonceNew)
	if pendingNonce < pendingNonceNew {
		return "", NonceToSmall
	}

	erc20Abi, err := abi.JSON(strings.NewReader(AbiErc20))
	if err != nil {
		return
	}

	input, err := erc20Abi.Pack("transfer", to, coins)
	if err != nil {
		return
	}

	msgGas := ethereum.CallMsg{
		From: from,
		To:   &contract,
		Data: input,
	}

	gasLimit, err := client.EstimateGas(ctx, msgGas)
	log.Log.Debug("gasLimit: ", gasLimit)

	if err != nil {
		return "", err
	}

	gas := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
	log.Log.Debug("gas: ", gas)

	if balance.Cmp(gas) <= 0 {
		return "", BalanceLessGasError
	}

	unsignedTx := types.NewTransaction(pendingNonce, contract, big.NewInt(0), gasLimit, gasPrice, input)
	signedTx, _ := types.SignTx(unsignedTx, types.NewEIP155Signer(big.NewInt(MainNetChainId)), privateKey)

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Log.Error("send transaction", err)
		return "", err
	}

	txHash := signedTx.Hash()
	log.Log.Debug("sendTx txHash:", txHash)

	hash = txHash.Hex()
	retryTimes := 0
	for {
		select {
		case <-time.NewTimer(MaxRetryTimeDurationSeconds * time.Second).C:
			if retryTimes >= MaxRetrySync {
				return hash, nil
			}

			receipt, err := client.TransactionReceipt(ctx, txHash)
			if err != nil {
				log.Log.Error("get transaction receipt: ", err)
				retryTimes++
				continue
			}

			if receipt.Status == types.ReceiptStatusSuccessful && receipt.BlockNumber != nil {
				log.Log.Debug("get transaction success status", " Tx In BlockNumber: ", receipt.BlockNumber, " GasUse: ", receipt.GasUsed, " Logs: ", receipt.Logs)
				return hash, nil
			}

			if receipt.Status == types.ReceiptStatusFailed {
				log.Log.Error("get transaction failed status")
				return "", SendTransactionFailedError
			}

		case <-ctx.Done():
			log.Log.Error("get transaction time out context")
			return "", ReadTransactionTimeOutError
		}
	}
}

func (j *Jk) SendContractAsync(ctx context.Context, senderPrivate string, receive string, amount float64, nonce uint64, contractAddr string) (hash string, err error) {
	if !common.IsHexAddress(receive) {
		return "", NotAnHexAddress
	}

	if contractAddr == "" {
		return "", ContractNotEmpty
	}

	client, err := j.Acquire()
	if err != nil {
		return
	}
	defer j.Release(client)

	if strings.HasPrefix(senderPrivate, "0x") {
		senderPrivate = senderPrivate[2:]
	}
	privateKey, err := crypto.HexToECDSA(senderPrivate)
	if err != nil {
		log.Log.Error(fmt.Sprintf("recover key err: %v", err))
		return "", PrivateKeyError
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Log.Error("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return "", PrivateKeyError
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	log.Log.Debug("Public Key: ", hexutil.Encode(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	log.Log.Debug("Address: ", address)

	balanceContract, decimals, err := j.GetBalanceOfContract(ctx, address, contractAddr)
	if err != nil {
		return "", err
	}

	if balanceContract < amount {
		return "", BalanceLessAmountError
	}

	from := common.HexToAddress(address)
	to := common.HexToAddress(receive)
	contract := common.HexToAddress(contractAddr)

	strAmount := fmt.Sprintf("%f", amount)
	f, success := new(big.Float).SetString(strAmount)
	if !success {
		return "", AmountError
	}

	fCoins := new(big.Float).Mul(f, big.NewFloat(decimals))
	coins, _ := fCoins.Int(nil)

	log.Log.Debug("from: ", from, " to: ", to, " contractAddr: ", contractAddr, " coins: ", coins)

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Log.Error("get balance err: ", err)
		return "", err
	}
	log.Log.Debug("gasPrice: ", gasPrice)

	balance, err := client.BalanceAt(ctx, from, nil)
	if err != nil {
		log.Log.Error("get balance err: ", err)
		return "", err
	}
	log.Log.Debug("balance: ", balance)

	var pendingNonce uint64
	if nonce == 0 {
		pendingNonce, err = client.PendingNonceAt(ctx, from)
		if err != nil {
			log.Log.Error("get pendingNonce err: ", err)
			return "", err
		}
	} else {
		pendingNonce = nonce
	}
	log.Log.Debug("pendingNonce: ", pendingNonce)

	erc20Abi, err := abi.JSON(strings.NewReader(AbiErc20))
	if err != nil {
		return
	}

	input, err := erc20Abi.Pack("transfer", to, coins)
	if err != nil {
		return
	}

	msgGas := ethereum.CallMsg{
		From: from,
		To:   &contract,
		Data: input,
	}

	gasLimit, err := client.EstimateGas(ctx, msgGas)
	log.Log.Debug("gasLimit: ", gasLimit)

	if err != nil {
		return "", err
	}

	gas := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
	log.Log.Debug("gas: ", gas)

	if balance.Cmp(gas) <= 0 {
		return "", BalanceLessGasError
	}

	unsignedTx := types.NewTransaction(pendingNonce, contract, big.NewInt(0), gasLimit, gasPrice, input)
	signedTx, _ := types.SignTx(unsignedTx, types.NewEIP155Signer(big.NewInt(MainNetChainId)), privateKey)

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Log.Error("send transaction", err)
		return "", err
	}

	txHash := signedTx.Hash()
	log.Log.Debug("sendTx txHash:", txHash)

	hash = txHash.Hex()
	return
}

func (j *Jk) SendContractInputDataSync(ctx context.Context, senderPrivate string, inputData []byte, contractAddr string) (hash string, err error) {
	client, err := j.Acquire()

	if err != nil {
		return
	}
	defer j.Release(client)

	if strings.HasPrefix(senderPrivate, "0x") {
		senderPrivate = senderPrivate[2:]
	}
	privateKey, err := crypto.HexToECDSA(senderPrivate)
	if err != nil {
		log.Log.Error(fmt.Sprintf("recover key err: %v", err))
		return "", PrivateKeyError
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Log.Error("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return "", PrivateKeyError
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	log.Log.Debug("Public Key: ", hexutil.Encode(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	log.Log.Debug("Address: ", address)

	from := common.HexToAddress(address)

	balance, err := client.BalanceAt(ctx, from, nil)
	if err != nil {
		return "", err
	}
	log.Log.Debug("balance: ", balance)

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Log.Error("get balance err: ", err)
		return "", err
	}
	log.Log.Debug("gasPrice: ", gasPrice)

	to := common.HexToAddress(contractAddr)
	msg := ethereum.CallMsg{
		From: from,
		To:   &to,
		Data: inputData,
	}

	gasLimit, err := client.EstimateGas(ctx, msg)
	if err != nil {
		log.Log.Error("EstimateGas: ", err)
		return "", err
	}

	//100000000000 * 10000000
	log.Log.Debug("gasLimit: ", gasLimit)

	gas := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
	log.Log.Debug("gas: ", gas)

	if balance.Cmp(gas) <= 0 {
		return "", errors.New(fmt.Sprintf("gas not enough, gas: %v balance: %v", gas.String(), balance.String()))
	}

	pendingNonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", err
	}

	log.Log.Debug("pendingNonce: ", pendingNonce)
	unsignedTx := types.NewTransaction(pendingNonce, to, big.NewInt(0), gasLimit, gasPrice, inputData)
	signedTx, err := types.SignTx(unsignedTx, types.NewEIP155Signer(big.NewInt(MainNetChainId)), privateKey)
	if err != nil {
		return "", err
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Log.Error("send transaction", err)
		return "", err
	}

	txHash := signedTx.Hash()
	log.Log.Debug("sendTx txHash:", txHash)

	hash = txHash.Hex()
	retryTimes := 0
	for {
		select {
		case <-time.NewTimer(MaxRetryTimeDurationSeconds * time.Second).C:
			if retryTimes >= MaxRetrySync {
				return hash, nil
			}

			receipt, err := client.TransactionReceipt(ctx, txHash)
			if err != nil {
				log.Log.Error("get transaction receipt: ", err)
				retryTimes++
				continue
			}

			if receipt.Status == types.ReceiptStatusSuccessful && receipt.BlockNumber != nil {
				log.Log.Debug("get transaction success status", " Tx In BlockNumber: ", receipt.BlockNumber, " GasUse: ", receipt.GasUsed, " Logs: ", receipt.Logs)
				return hash, nil
			}

			if receipt.Status == types.ReceiptStatusFailed {
				log.Log.Error("get transaction failed status")
				return "", SendTransactionFailedError
			}

		case <-ctx.Done():
			log.Log.Error("get transaction time out context")
			return "", ReadTransactionTimeOutError
		}
	}
}

func (j *Jk) IsContract(ctx context.Context, address string) (bool, error) {
	client, err := j.Acquire()
	if err != nil {
		return false, err
	}

	defer j.Release(client)

	at, err := client.CodeAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return false, err
	}

	if string(at) != "" {
		return true, nil
	}

	return false, nil
}

func (j *Jk) StartScan(startNumber uint64, timeInternal time.Duration, handle func(tx *types.Transaction, block *types.Block) error) error {
	if startNumber < 1 {
		return errors.New("start number can not < 1")
	}

	defer j.Close()
	latestNumber := uint64(0)

	mutex := sync.Mutex{}
	for {
		select {
		case <-time.NewTimer(timeInternal).C:
			err := j.ExecuteBlocks(&mutex, latestNumber, startNumber, handle)
			if err != nil {
				log.Log.Error("ExecuteBlocks err", err, "continues")
				continue
			}
		}
	}
}

func (j *Jk) ExecuteBlocks(mutex *sync.Mutex, startNumber, latestNumber uint64, handle func(tx *types.Transaction, block *types.Block) error) error {

	mutex.Lock()
	defer func() {
		log.Log.Debug("defer unlock")
		mutex.Unlock()
	}()

	client, err := j.Acquire()
	defer func() {
		log.Log.Debug("defer Release")
		j.Release(client)
	}()

	if err != nil {
		log.Log.Error("acquire client: ", err)
		return err
	}

	highestNumber, err := client.BlockNumber(context.Background())
	if err != nil {
		log.Log.Error("get latest block number: ", err)
		return err
	}

	latestNumber = j.readLatestNumber(highestNumber)
	if startNumber > latestNumber {
		latestNumber = startNumber
	}

	log.Log.Debug("startNumber: ", startNumber, " highestNumber: ", highestNumber, " latestNumber: ", latestNumber)

	diffHeight := highestNumber - latestNumber
	if diffHeight <= 0 {
		log.Log.Debug("diffHeight <= 0: ", diffHeight, " it may no new block or block rollback, stop execute")
		return err
	}

	log.Log.Debug("diffHeight: ", diffHeight)

	for height := latestNumber + 1; height < latestNumber+diffHeight; height++ {
		time.Sleep(1 * time.Second)

		var block *types.Block
		block, err = client.BlockByNumber(context.Background(), big.NewInt(int64(height)))

		log.Log.Debug("==============handle start height: ", height, "================================")

		if err != nil {
			log.Log.Error("get block height: ", height, " happened, it may lost height, stop now. ", " err info: ", err)
			return err
		}

		transactions := block.Transactions()
		for _, tx := range transactions {
			log.Log.Debug("notify handle tx for business", " txHash: ", tx.Hash(), " blockNumber: ", block.Number())

			err = handle(tx, block)

			if err != nil {
				log.Log.Error("handle tx: ", tx.Hash(), " err happened ", "record block and to next")

				err = j.writeErrorTx(tx, block)
				if err != nil {
					log.Log.Error("write tx err info: ", tx, " err happened ", "record block and to next")
				}
			}
		}

		err = j.writeLatestNumber(height)
		if err != nil {
			log.Log.Error("write latest number: ", err)
			return err
		}

		log.Log.Debug("==============handle end height: ", height, "================================")
	}

	return nil
}

func (j *Jk) GetTransactionReceiptByHash(ctx context.Context, hash string) (*types.Receipt, error) {
	client, err := j.Acquire()
	if err != nil {
		return nil, err
	}
	defer j.Release(client)

	tx, err := client.TransactionReceipt(ctx, common.HexToHash(hash))
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (j *Jk) GetTransactionByHash(ctx context.Context, hash string) (*types.Transaction, bool, error) {
	client, err := j.Acquire()
	if err != nil {
		return nil, false, err
	}
	defer j.Release(client)

	tx, pending, err := client.TransactionByHash(ctx, common.HexToHash(hash))
	if err != nil {
		return nil, pending, err
	}

	return tx, false, nil
}

func (j *Jk) GetBlockByHash(ctx context.Context, hash string) (*types.Block, error) {
	client, err := j.Acquire()
	if err != nil {
		return nil, err
	}
	defer j.Release(client)

	block, err := client.BlockByHash(ctx, common.HexToHash(hash))
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (j *Jk) writeErrorTx(tx *types.Transaction, block *types.Block) error {

	j.rw.Lock()
	defer j.rw.Unlock()

	var open *os.File
	var path = "./errtx.info"

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		open, err = os.Create(path)
		if err != nil {
			log.Log.Error("create errtx.info: ", err)
		}
	} else {
		open, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Log.Error("open errtx.info: ", err)
		}
	}

	defer open.Close()

	msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId()), block.BaseFee())
	if err != nil {
		log.Log.Error("decode to message: ", err)
		return err
	}

	height := block.Number()
	hash := tx.Hash()
	value := tx.Value()
	to := tx.To()
	nonce := tx.Nonce()
	data := tx.Data()
	from := msg.From()

	log.Log.Debug(fmt.Sprintf("from: %v hash: %v value: %v  to: %v nonce: %v  data: %v height: %v", from, hash, value, to, nonce, data, height))

	_, err = open.WriteString(fmt.Sprintf("from: %v hash: %v value: %v  to: %v nonce: %v  data: %v height: %v", from, hash, value, to, nonce, data, height) + "\n")
	if err != nil {
		log.Log.Error("write to errtx.info: ", err)
		return err
	}

	return nil
}

func (j *Jk) writeLatestNumber(latestNumber uint64) error {

	j.rw.Lock()
	defer j.rw.Unlock()

	var open *os.File
	path := "./latestNumber.info"
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		open, err = os.Create(path)
		if err != nil {
			log.Log.Error("create latestNumber.info: ", err)
		}
	} else {
		open, err = os.OpenFile(path, os.O_RDWR, os.ModeAppend)
		if err != nil {
			log.Log.Error("open latestNumber.info: ", err)
		}
	}

	defer open.Close()

	formatUint := strconv.FormatUint(latestNumber, 10)
	_, err = open.Write([]byte(formatUint))
	if err != nil {
		log.Log.Error("write latestNumber.info: ", err)
		return err
	}

	return nil
}

func (j *Jk) readLatestNumber(highestNumber uint64) uint64 {
	j.rw.Lock()
	defer j.rw.Unlock()

	var open *os.File
	path := "./latestNumber.info"
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		open, err = os.Create(path)
		if err != nil {
			log.Log.Error("create latestNumber.info: ", err)
		}
	} else {
		open, err = os.OpenFile(path, os.O_RDWR, os.ModeAppend)
		if err != nil {
			log.Log.Error("open latestNumber.info: ", err)
		}
	}

	defer open.Close()

	all, err := ioutil.ReadAll(open)
	if err != nil {
		log.Log.Error("read latestNumber.info: ", err)
		return highestNumber
	}

	a := strings.TrimSpace(string(all))
	if a == "" {
		return 0
	}

	f, success := new(big.Float).SetString(a)
	if !success {
		log.Log.Error("float set string latestNumber.info: ", err)
		return highestNumber
	}

	u, _ := f.Uint64()
	return u
}
