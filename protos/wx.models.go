package protos

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

	Code string `json:"code,omitempty"`

	UserId     string `json:"uid,omitempty"`
	Openid     string `json:"openid,omitempty"`
	SessionKey string `json:"session_key,omitempty"`
	ExpiresIn  int    `json:"expires_in,omitempty"`
	LoginAt    int64  `json:"login_at,omitempty"`
}

///////////////////////////////////////////////////////////////////////////////
// 请求参数
///////////////////////////////////////////////////////////////////////////////
type WxMiniAppLoginReq struct {
	Code string `json:"code" validate:"required"`
}
