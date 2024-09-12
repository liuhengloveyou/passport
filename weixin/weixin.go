package weixin

import (
	"fmt"

	"github.com/bytedance/sonic"
	gocommon "github.com/liuhengloveyou/go-common"
	"go.uber.org/zap"

	"github.com/liuhengloveyou/passport/common"
)

type WeixinErrResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}

/*
access_token	网页授权接口调用凭证,注意：此access_token与基础支持的access_token不同
expires_in	access_token接口调用凭证超时时间，单位（秒）
refresh_token	用户刷新access_token
openid	用户唯一标识，请注意，在未关注公众号时，用户访问公众号的网页，也会产生一个用户和公众号唯一的OpenID
scope	用户授权的作用域，使用逗号（,）分隔
is_snapshotuser	是否为快照页模式虚拟账号，只有当用户是快照页模式虚拟账号时返回，值为1
unionid	用户统一标识（针对一个微信开放平台账号下的应用，同一用户的 unionid 是唯一的），只有当scope为"snsapi_userinfo"时返回
*/
type WeixinMpAccessTokenResponse struct {
	WeixinErrResponse

	AccessToken    string `json:"access_token"`
	ExpiresIn      int    `json:"expires_in"`
	RefreshToken   string `json:"refresh_token"`
	OpenId         string `json:"openid"`
	Scope          string `json:"scope"`
	IsSnapshotuser int    `json:"is_snapshotuser"`
	Unionid        string `json:"unionid"`
}

type WeixinMpUserInfoResponse struct {
	WeixinErrResponse

	OpenId     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int64    `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	Headimgurl string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	Unionid    string   `json:"unionid"`
}

func GetAccessToken(appId, appSecret string, code string) (accessToken *WeixinMpAccessTokenResponse, err error) {
	apiAddr := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		appId, appSecret, code)
	resp, body, err := gocommon.GetRequest(apiAddr, nil)
	if err != nil {
		return nil, err
	}
	common.Logger.Info("weixin.GetAccessToken: ", zap.String("appid", appId), zap.String("code", code), zap.String("body", string(body)))

	if resp.StatusCode != 200 {
		common.Logger.Error("weixin.GetAccessToken http ERR: ", zap.String("appid", appId), zap.String("code", code), zap.Any("resp", resp))
		return nil, common.ErrWxService
	}

	wxResp := &WeixinMpAccessTokenResponse{}
	if err = sonic.Unmarshal(body, wxResp); err != nil {
		common.Logger.Error("weixin.GetAccessToken json ERR: ", zap.String("appid", appId), zap.String("code", code), zap.Error(err))
		return nil, common.ErrWxService
	}

	if wxResp.ErrCode != 0 {
		common.Logger.Error("weixin.GetAccessToken resp ERR: ", zap.String("appid", appId), zap.String("code", code), zap.Any("wxresp", wxResp))
		return
	}

	return wxResp, nil
}

func GetUserInfo(accessToken, openId string) (userInfo *WeixinMpUserInfoResponse, err error) {
	apiAddr := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN", accessToken, openId)

	resp, body, err := gocommon.GetRequest(apiAddr, nil)
	if err != nil {
		return nil, err
	}
	common.Logger.Info("weixin.GetUserInfo: ", zap.String("openId", openId), zap.String("body", string(body)))

	if resp.StatusCode != 200 {
		common.Logger.Error("weixin.GetUserInfo http ERR: ", zap.String("openId", openId), zap.Any("resp", resp))
		return nil, common.ErrWxService
	}

	wxResp := &WeixinMpUserInfoResponse{}
	if err = sonic.Unmarshal(body, wxResp); err != nil {
		common.Logger.Error("weixin.GetUserInfo json ERR: ", zap.String("openId", openId), zap.Error(err))
		return nil, common.ErrWxService
	}

	if wxResp.ErrCode != 0 {
		common.Logger.Error("weixin.GetUserInfo resp ERR: ", zap.String("openId", openId), zap.Any("wxresp", wxResp))
		return
	}

	return wxResp, nil
}
