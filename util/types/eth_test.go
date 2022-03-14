package types

import "testing"

func TestGenerateKey(t *testing.T) {

	key, privateKey, err := GenerateKey()
	if err != nil {
		t.Error(err)
	}

	t.Log(key, privateKey)
}
