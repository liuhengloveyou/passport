package service

import (
	"encoding/json"
	"fmt"

	gocommon "github.com/liuhengloveyou/go-common"
)

const jscode2session = "https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code"

type MiniAppErr struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type LoginRst struct {
	ErrMsg string `json:"errMsg"`
	Code   string `json:"code"`
}

type MiniAppUserInfo struct {
	sessionid string

	UserId     string `json:"uid"`
	Code       string `json:"code"`
	Openid     string `json:"openid"`
	SessionKey string `json:"session_key"`

	MiniAppErr
}

type MiniApp struct {
	Code       string
	AppID      string
	AppSecrect string
}

func (p *MiniApp) Login() (*MiniAppUserInfo, error) {
	_, wxbody, e := gocommon.GetRequest(fmt.Sprintf(jscode2session, p.AppID, p.AppSecrect, p.Code), nil)
	if e != nil {
		return nil, e
	}

	userInfo := &MiniAppUserInfo{}
	if e = json.Unmarshal(wxbody, userInfo); e != nil {
		return nil, e
	}

	if userInfo.ErrCode != 0 && userInfo.ErrMsg != "" {
		return nil, fmt.Errorf("jscode2session ERR: %v, %v", userInfo.ErrCode, userInfo.ErrMsg)
	}

	return userInfo, nil
}
