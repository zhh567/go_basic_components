package go_web_test

import (
	"testing"
)

func TestMain(t *testing.T) {

}

func Test_tmp(t *testing.T) {
	var ss []string
	t.Log(ss)
	ss = append(ss, "xx")
	t.Log(ss)
}
