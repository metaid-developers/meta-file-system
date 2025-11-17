package tool

import (
	"fmt"
	"testing"
)

func TestGetToday0Time(t *testing.T) {
	fmt.Println(GetToday0Time())
	fmt.Println(GetToday24Time())
}

func TestMakeTimestampV2(t *testing.T) {
	date := "2024-08-26-00-00-00"
	fmt.Println(MakeTimestampV2(date))
}
