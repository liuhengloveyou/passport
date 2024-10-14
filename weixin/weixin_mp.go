package weixin

import (
	"fmt"

	"github.com/bytedance/sonic"
	gocommon "github.com/liuhengloveyou/go-common"
	"go.uber.org/zap"

	"github.com/liuhengloveyou/passport/common"
)

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
