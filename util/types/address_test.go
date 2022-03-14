package types

import (
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestAccAddress_HexString1(t *testing.T) {
	addr, err := AccAddressFromHex("0xEC5449e1719a4f3555Ef71A98706BDBbFbaCA2C5")
	if err != nil {
		t.Error(err)
	}

	t.Log(addr.Bech32String("fb"))
}

func TestAccAddress_HexString(t *testing.T) {
	addr, err := AccAddressFromHex("EC5449e1719a4f3555Ef71A98706BDBbFbaCA2C5")
	if err != nil {
		t.Error(err)
	}

	t.Log(addr.Bech32String("fb"))
}

func TestAccAddress_FromBech32String(t *testing.T) {
	addr, err := AccAddressFromBech32("fb1a32ynct3nf8n2400wx5cwp4ah0a6egk9lzq2dl")
	if err != nil {
		t.Error(err)
	}

	t.Log(addr, common.BytesToAddress(addr))
}
