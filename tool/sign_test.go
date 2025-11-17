package tool

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"log"
	"testing"
)

func TestSignMessage(t *testing.T) {
	privateKeyHex := "8170940a65bda743704be89096ce6d292f052dbb897f4b7aa5d92aa1d0e64531"
	message := "a47b2012afd6b72f5fd869acca960fe25bca772d69482e8c3ada70fd75b3a5e0i0"
	sig, err := SignMessage(message, privateKeyHex)
	if err != nil {
		t.Errorf("SignMessage() failed, err: %v", err)
		return
	}
	t.Logf("SignMessage() sig: %v", sig)
}

func TestVerifySign(t *testing.T) {
	privateKeyHex := "8170940a65bda743704be89096ce6d292f052dbb897f4b7aa5d92aa1d0e64531"
	publicKey := ""
	messageSign := "3044022033fec8915ffdf1f6f068ba580a1b310fdc5f32eaac784667a0f6e2feca58a4b3022006b296a807e35fb90578a026873a744540eb49e8686851617e7c1049c20ac497"
	message := "a47b2012afd6b72f5fd869acca960fe25bca772d69482e8c3ada70fd75b3a5e0i0"

	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		log.Fatal(err)
	}
	privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	publicKey = hex.EncodeToString(privateKey.PubKey().SerializeCompressed())
	fmt.Println("publicKey: ", publicKey)

	verified, err := VerifySign(message, messageSign, publicKey)
	if err != nil {
		t.Errorf("VerifySign() failed, err: %v", err)
		return
	}
	t.Logf("VerifySign() verified: %v", verified)
}
