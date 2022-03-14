package blx

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"testing"
)

func TestGetBalance(t *testing.T) {

	address := "0xFd2955B33Fa3bE18b6ef3a90097F8a25F5E5FF85"
	jk := NewJk(2)
	of, err := jk.GetBalanceOf(context.Background(), address)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestGetBalanceOfContract(t *testing.T) {

	address := "0x116663f85a8727410efa33f7051265efae77ed98"
	contractAddr := "0x03007fcaa04cec04820ed54e1a49b2e0f69cc298"
	jk := NewJk(2)
	of, _, err := jk.GetBalanceOfContract(context.Background(), address, contractAddr)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestJk_SendSync(t *testing.T) {

	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"
	jk := NewJk(2)
	of, err := jk.SendSync(context.Background(), senderPrivate, to, 1.1)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestJk_SendAsync(t *testing.T) {

	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"

	jk := NewJk(2)
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

	jk := NewJk(2)
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

	jk := NewJk(2)
	of, err := jk.SendContractAsync(context.Background(), senderPrivate, to, 1.1, 0, contractAddr)
	if err != nil {
		t.Error(err)
	}

	t.Log(of)
}

func TestSendContractSyncBatch(t *testing.T) {
	senderPrivate := ""
	to := "0x6cAa27dFc890d772B5fA3dB3dAaa39Bf576DC109"
	contractAddr := "0x03007fcaa04cec04820ed54e1a49b2e0f69cc298"

	jk := NewJk(2)

	for i := 0; i < 10; i++ {
		of, err := jk.SendContractSync(context.Background(), senderPrivate, to, 1.1, contractAddr)
		if err != nil {
			t.Error(err)
		}

		t.Log(of)
	}

}

func TestSendContractAsyncBatch(t *testing.T) {

	jk := NewJk(2)

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

	jk := NewJk(2)
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
	jk := NewJk(2)
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
