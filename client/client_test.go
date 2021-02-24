package client_test

import (
	"fmt"
	"testing"

	"github.com/liuhengloveyou/passport/client"

	gocommon "github.com/liuhengloveyou/go-common"
)

var passClient = &client.Passport{"http://127.0.0.1:10001"}

func TestUserAdd(t *testing.T) {
	rst, e := passClient.UserAdd("18510511015", "liuhengloveyou@gmail.com", "L", "123456")
	fmt.Println(rst, e)
}

func TestMiniAppLogin(t *testing.T) {
	_, body, err := gocommon.PostRequest("http://127.0.0.1:10001/miniapp/login", nil, nil, []byte("{\"code\":\"code.....\"}"))

	fmt.Println(string(body), err)
}

func TestUserAuth(t *testing.T) {
	rst, e := passClient.UserAuth("2e7540034d62a6c1d535cbe9d434f59b3f8c73b7a7c341ec")
	fmt.Println(string(rst), e)

}
