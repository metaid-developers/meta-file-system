package tool

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

func StrToGzip(data string) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(data)); err != nil {
		return nil, err
	}
	if err := gz.Flush(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	//fmt.Println(b)
	return b.Bytes(), nil
}

func GzipToStr(data []byte) (string, error) {
	rdata := bytes.NewReader(data)
	r, err := gzip.NewReader(rdata)
	if err != nil {
		return "", err
	}
	s, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(s), nil
}
