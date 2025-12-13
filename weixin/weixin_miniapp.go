package weixin

import (
	"encoding/json"
	"fmt"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/common"
)

/*
session_key	string	会话密钥
unionid	string	用户在开放平台的唯一标识符，若当前小程序已绑定到微信开放平台账号下会返回，详见 UnionID 机制说明。
errmsg	string	错误信息
openid	string	用户唯一标识
errcode	int32	错误码
*/
type WeixinJscode2sessionResponse struct {
	WeixinErrResponse

	Code       string `json:"code"`
	OpenId     string `json:"openid"`
	Unionid    string `json:"unionid"`
	SessionKey string `json:"session_key"`
}

// https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/code2Session.html
func WxMiniAppLogin(code string, appId, appSecret string) (*WeixinJscode2sessionResponse, error) {
	const jscode2session = "https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code"

	_, body, e := gocommon.GetRequest(fmt.Sprintf(jscode2session, appId, appSecret, code), nil)
	if e != nil {
		common.Logger.Sugar().Errorf("jscode2session ERR: %v\n", e)
		return nil, e
	}
	common.Logger.Sugar().Infof("jscode2session resp: %v\n", string(body))

	resp := &WeixinJscode2sessionResponse{}
	if e = json.Unmarshal(body, &resp); e != nil {
		common.Logger.Sugar().Errorf("jscode2session json ERR: %v\n", e)
		return nil, e
	}

	if resp.ErrCode != 0 && resp.ErrMsg != "" {
		common.Logger.Sugar().Errorf("jscode2session resp ERR: %v\n", resp)
		return nil, common.ErrWxService
	}
	resp.Code = code

	common.Logger.Sugar().Errorf("jscode2session OK: %v\n", resp)
	return resp, nil
}

func WxMiniAppUserInfoUpdate(req WxMiniAppUserInfoUpdateReq) {
	//
	//rows, e := dao.UserUpdateExt(uid, &userInfo.Ext)
	//if e != nil {
	//	common.Logger.Sugar().Errorf("TenantUpdateUserExt ERR: %v", e)
	//	return common.ErrService
	//}
	//if rows < 1 {
	//	common.Logger.Sugar().Warnf("TenantUpdateUserExt RowsAffected 0")
	//}
	//
	//return nil

}
