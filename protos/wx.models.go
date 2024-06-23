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
	Openid     string `json:"openid,omitempty"`
	SessionKey string `json:"session_key,omitempty"`
	ExpiresIn  int    `json:"expires_in,omitempty"`
	LoginAt    int64  `json:"login_at,omitempty"`

	UserId             string `json:"uid,omitempty"`
	Avatar             string `json:"avatar"`
	Province           string `json:"province"`
	City               string `json:"city"`
	NickName           string `json:"nick_name"`
	IsStudentCertified string `json:"is_student_certified"`
	UserType           string `json:"user_type"`
	UserStatus         string `json:"user_status"`
	IsCertified        string `json:"is_certified"`
	Gender             string `json:"gender"`
}

func (p *MiniAppSessionInfo) UserKey() string {
	if p.UserId != "" {
		return p.UserId // 支付宝
	} else if p.Openid != "" {
		return p.Openid // 微信
	}

	return ""
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
