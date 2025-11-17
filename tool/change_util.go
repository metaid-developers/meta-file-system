package tool

import "strconv"

func StrToInt64(str string) int64 {
	data, _ := strconv.ParseInt(str, 10, 64)
	return data
}

func ByteArrayToIntArray(data []byte) []int {
	var (
		dataIntArray []int = make([]int, 0)
	)
	for _, v := range data {
		dataIntArray = append(dataIntArray, int(v))
	}
	return dataIntArray
}
