package multisig

import (
	amino "github.com/tendermint/go-amino"

	"github.com/zhengjianfeng1103/FbSdk/libs/crypto"
	"github.com/zhengjianfeng1103/FbSdk/libs/crypto/ed25519"
	"github.com/zhengjianfeng1103/FbSdk/libs/crypto/secp256k1"
	"github.com/zhengjianfeng1103/FbSdk/libs/crypto/sr25519"
)

// TODO: Figure out API for others to either add their own pubkey types, or
// to make verify / marshal accept a cdc.
const (
	PubKeyMultisigThresholdAminoRoute = "tendermint/PubKeyMultisigThreshold"
)

var cdc = amino.NewCodec()

func init() {
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(PubKeyMultisigThreshold{},
		PubKeyMultisigThresholdAminoRoute, nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(sr25519.PubKeySr25519{},
		sr25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)
}

func RegisterKeyType(o interface{}, name string) {
	cdc.RegisterConcrete(o, name, nil)
}
