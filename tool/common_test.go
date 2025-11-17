package tool

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestParseDataUrl(t *testing.T) {
	dataUrl := "eyJwIjoiYnJjLTIwIiwib3AiOiJ0cmFuc2ZlciIsInRpY2siOiJPUlhDIiwiYW10IjoiMTAwMCJ9"
	fmt.Println(string(ParseDataUrl(dataUrl)))
}

func TestGenerateAESKey(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(hex.EncodeToString(key))
}
