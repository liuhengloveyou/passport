package weixin

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

// type MiniAppSessionInfo struct {
// 	MiniAppErr

// 	Code       string `json:"code,omitempty"`
// 	SessionKey string `json:"session_key,omitempty"`
// 	ExpiresIn  int    `json:"expires_in,omitempty"`
// 	LoginAt    int64  `json:"login_at,omitempty"`

// 	// weixin拉取用户信息(需scope为 snsapi_userinfo)
// 	Openid     string `json:"openid,omitempty"`
// 	NickName   string `json:"nickname"`
// 	Sex        int    `json:"sex"`
// 	Province   string `json:"province"`
// 	City       string `json:"city"`
// 	Country    string `json:"country"`
// 	HeadImgUrl string `json:"headimgurl"`

// 	//
// 	UserId             string `json:"uid,omitempty"`
// 	Avatar             string `json:"avatar"`
// 	IsStudentCertified string `json:"is_student_certified"`
// 	UserType           string `json:"user_type"`
// 	UserStatus         string `json:"user_status"`
// 	IsCertified        string `json:"is_certified"`
// 	Gender             string `json:"gender"`
// }

// func (p *MiniAppSessionInfo) UserKey() string {
// 	// 微信/支付宝都用openid.
// 	// 但是支付宝的太长, 截断先

// 	key := p.Openid
// 	if len(key) > 45 {
// 		key = key[:45]
// 	}

// 	return key

// }

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
