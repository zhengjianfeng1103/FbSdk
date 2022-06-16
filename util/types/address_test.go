package types

import (
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestAccAddress_HexString1(t *testing.T) {
	addr, err := AccAddressFromHex("25d91BcE0D59C992C3382F775C48d8a1dc217569")
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

	//fb1yhv3hnsdt8ye9sec9am4cjxc58wzzatfacgmrq
	addr, err := AccAddressFromBech32("fblyhv3hnsdt8ye9sec9am4cjxc58wzzatfacgmrq")
	if err != nil {
		t.Error(err)
	}

	t.Log(addr, common.BytesToAddress(addr))
}
