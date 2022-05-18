package types

import (
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestAccAddress_HexString1(t *testing.T) {
	addr, err := AccAddressFromHex("116663f85A8727410EFA33F7051265EFAE77ed98")
	if err != nil {
		t.Error(err)
	}

	t.Log(addr.Bech32String("fb"))
}

func TestAccAddress_HexString(t *testing.T) {
	addr, err := AccAddressFromHex("6caa27Dfc890d772b5Fa3Db3DaaA39bF576dC109")
	if err != nil {
		t.Error(err)
	}

	t.Log(addr.Bech32String("fb"))
}

func TestAccAddress_FromBech32String(t *testing.T) {
	addr, err := AccAddressFromBech32("fb1jyzrwcfg7p6jq22wf7qnez5lzq0cap2nzgi90y")
	if err != nil {
		t.Error(err)
	}

	t.Log(addr, common.BytesToAddress(addr))
}
