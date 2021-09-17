package service

import (
	"encoding/json"
	"fmt"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"time"

	gocommon "github.com/liuhengloveyou/go-common"
)


type miniAppService struct {
	AppID     string
	AppSecret string
}

func (p *miniAppService) Login(code string) (*protos.MiniAppSessionInfo, error) {
	const jscode2session = "https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code"

	_, body, e := gocommon.GetRequest(fmt.Sprintf(jscode2session, p.AppID, p.AppSecret, code), nil)
	if e != nil {
		common.Logger.Sugar().Errorf("jscode2session ERR: %v\n", e)
		return nil, e
	}
	common.Logger.Sugar().Infof("jscode2session resp: %v\n", string(body))

	sessionInfo := &protos.MiniAppSessionInfo{
		LoginAt: time.Now().Unix(),
	}
	if e = json.Unmarshal(body, sessionInfo); e != nil {
		common.Logger.Sugar().Errorf("jscode2session json ERR: %v\n", e)
		return nil, e
	}

	if sessionInfo.ErrCode != 0 && sessionInfo.ErrMsg != "" {
		common.Logger.Sugar().Errorf("jscode2session resp ERR: %v\n", sessionInfo)
		return nil, common.ErrWxService
	}
	sessionInfo.Code = code

	common.Logger.Sugar().Errorf("jscode2session OK: %v\n", sessionInfo)
	return sessionInfo, nil
}


func (p *miniAppService) WxMiniAppUserInfoUpdate(req protos.WxMiniAppUserInfoUpdateReq) {
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

