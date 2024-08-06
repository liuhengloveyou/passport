package protos

const MiniAppSessionInfoKey = "MiniAppSessionInfo"

type MiniAppErr struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}

type LoginRst struct {
	ErrMsg string `json:"errMsg"`
	Code   string `json:"code"`
}

type MiniAppSessionInfo struct {
	MiniAppErr

	Code       string `json:"code,omitempty"`
	SessionKey string `json:"session_key,omitempty"`
	ExpiresIn  int    `json:"expires_in,omitempty"`
	LoginAt    int64  `json:"login_at,omitempty"`

	// weixin拉取用户信息(需scope为 snsapi_userinfo)
	Openid     string `json:"openid,omitempty"`
	NickName   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	HeadImgUrl string `json:"headimgurl"`

	//
	UserId             string `json:"uid,omitempty"`
	Avatar             string `json:"avatar"`
	IsStudentCertified string `json:"is_student_certified"`
	UserType           string `json:"user_type"`
	UserStatus         string `json:"user_status"`
	IsCertified        string `json:"is_certified"`
	Gender             string `json:"gender"`
}

func (p *MiniAppSessionInfo) UserKey() string {
	// 微信/支付宝都用openid.
	// 但是支付宝的太长, 截断先

	key := p.Openid
	if len(key) > 45 {
		key = key[:40]
	}

	return key

}

// /////////////////////////////////////////////////////////////////////////////
// 请求参数
// /////////////////////////////////////////////////////////////////////////////
type WxMiniAppLoginReq struct {
	Code string `json:"code" validate:"required"`
}

type WxMiniAppUserInfoUpdateReq struct {
	AvatarUrl string `json:"avatarUrl" validate:"required"`
	City      string `json:"city"`
	Country   string `json:"country"`
	Gender    int    `json:"gender"`
	NickName  string `json:"nickName"`
	Province  string `json:"province"`
}
