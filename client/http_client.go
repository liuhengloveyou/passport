package client

import (
	"encoding/json"
	"fmt"
	"github.com/liuhengloveyou/passport/protos"
	"net/http"

	validator "github.com/go-playground/validator/v10"
	gocommon "github.com/liuhengloveyou/go-common"
)

type Passport struct {
	ServAddr string
}

func (p *Passport) UserAdd(cellphone, email, nickname, password string) (userid string, err error) {
	userinfo := &protos.UserReq{Cellphone: cellphone, Email: email, Nickname: nickname, Password: password}
	if err = validator.New().Struct(userinfo); err != nil {
		return "", err
	}

	body, err := json.Marshal(userinfo)
	if err != nil {
		return "", err
	}

	response, resBody, err := gocommon.PostRequest(p.ServAddr+"/user/add", nil, nil, body)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", resBody)
	}

	rst := make(map[string]string, 0)
	if err = json.Unmarshal(resBody, &rst); err != nil {
		return "", err
	}

	return rst["userid"], nil
}

func (p *Passport) UserAuth(token string) (sessionInfo []byte, err error) {
	header := map[string]string{"TOKEN": token}
	response, body, err := gocommon.GetRequest(p.ServAddr+"/user/auth", header)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, nil
	}

	return body, nil
}
