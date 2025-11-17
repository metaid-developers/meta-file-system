package node

import (
	"fmt"
	"meta-file-system/conf"
	"testing"
)

func TestBasicAuth(t *testing.T) {
	username := "showpay"
	password := "showpay88.."
	fmt.Println(BasicAuth(username, password))
}

func TestGetDogeCurrentBlockHeight(t *testing.T) {
	height, err := CurrentBlockHeight("")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(height)
}

func TestGetTxDetail(t *testing.T) {
	conf.InitConfig()
	tx, err := GetTxDetail("testnet", "")
	if err != nil {
		t.Fatal(err)
	} else {
		fmt.Println(tx)
	}
}
