package sms

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/liuhengloveyou/passport/v4/common"
	"go.uber.org/zap"
)

// 阿里云验证码模板变量名（申请模板时须与此一致）
const aliyunVerifyCodeParam = "code"

type SmsAliyun struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	Endpoint        string

	UserAddTemplateID    string
	UserLoginTemplateID  string
	GetBackPwdTemplateID string
	WxBindTemplateID     string

	client *dysmsapi.Client
}

func init() {
	Register("aliyun", NewSmsAliyun)
}

func NewSmsAliyun(config map[string]interface{}) Sms {
	p := &SmsAliyun{
		AccessKeyID:          configStr(config, "access_key_id"),
		AccessKeySecret:      configStr(config, "access_key_secret"),
		SignName:             configStr(config, "sign_name"),
		Endpoint:             configStr(config, "endpoint"),
		UserAddTemplateID:    configStr(config, "user_add_template_id"),
		UserLoginTemplateID:  configStr(config, "user_login_template_id"),
		GetBackPwdTemplateID: configStr(config, "getback_pwd_template_id"),
		WxBindTemplateID:     configStr(config, "wx_bind_template_id"),
	}
	if p.Endpoint == "" {
		p.Endpoint = "dysmsapi.aliyuncs.com"
	}
	if p.AccessKeyID == "" || p.AccessKeySecret == "" {
		return nil
	}
	return p
}

func (p *SmsAliyun) SendUserAddSms(phoneNumber string, aliveSecond int64) (code string, err error) {
	return p.sendVerifyCode(phoneNumber, p.UserAddTemplateID)
}

func (p *SmsAliyun) SendUserLoginSms(phoneNumber string, aliveSecond int64) (code string, err error) {
	return p.sendVerifyCode(phoneNumber, p.UserLoginTemplateID)
}

func (p *SmsAliyun) SendGetBackPwdSms(phoneNumber string, aliveSecond int64) (code string, err error) {
	return p.sendVerifyCode(phoneNumber, p.GetBackPwdTemplateID)
}

func (p *SmsAliyun) SendWxBindSms(phoneNumber string, aliveSecond int64) (code string, err error) {
	return p.sendVerifyCode(phoneNumber, p.WxBindTemplateID)
}

func (p *SmsAliyun) sendVerifyCode(phoneNumber, templateCode string) (code string, err error) {
	code = fmt.Sprintf("%06v", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(1000000))
	if err = p.sendSms(phoneNumber, templateCode, []string{code}); err != nil {
		common.Logger.Error("sendVerifyCode: sendSms error", zap.Error(err))
		return "", err
	}
	common.Logger.Info("sendVerifyCode: sendSms success", zap.String("phoneNumber", phoneNumber), zap.String("templateCode", templateCode), zap.String("code", code))
	return code, nil
}

func (p *SmsAliyun) ensureClient() error {
	if p.client != nil {
		return nil
	}
	c, err := dysmsapi.NewClient(&openapi.Config{
		AccessKeyId:     tea.String(p.AccessKeyID),
		AccessKeySecret: tea.String(p.AccessKeySecret),
		Endpoint:        tea.String(p.Endpoint),
	})
	if err != nil {
		return err
	}
	p.client = c
	return nil
}

func (p *SmsAliyun) sendSms(phoneNumber, templateCode string, vals []string) error {
	if err := p.ensureClient(); err != nil {
		common.Logger.Error("sendSms: ensureClient error", zap.Error(err))
		return err
	}

	paramJSON, err := buildAliyunVerifyParam(vals)
	if err != nil {
		common.Logger.Error("sendSms: buildAliyunVerifyParam error", zap.Error(err))
		return err
	}

	req := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(normalizeAliyunPhone(phoneNumber)),
		SignName:      tea.String(p.SignName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(paramJSON),
	}

	resp, err := p.client.SendSms(req)
	if err != nil {
		common.Logger.Error("sendSms: SendSms error", zap.Error(err))
		return err
	}

	apiCode := ""
	msg := ""
	if resp != nil && resp.Body != nil {
		if resp.Body.Code != nil {
			apiCode = *resp.Body.Code
		}
		if resp.Body.Message != nil {
			msg = *resp.Body.Message
		}
	}
	if !strings.EqualFold(apiCode, "OK") {
		common.Logger.Error("sendSms: SendSms error", zap.String("apiCode", apiCode), zap.String("msg", msg))
		return fmt.Errorf("aliyun sms: %s %s", apiCode, msg)
	}
	common.Logger.Info("sendSms: SendSms success", zap.String("phoneNumber", phoneNumber), zap.String("templateCode", templateCode), zap.String("paramJSON", paramJSON))
	return nil
}

func buildAliyunVerifyParam(vals []string) (string, error) {
	if len(vals) == 0 {
		return "", nil
	}
	b, err := json.Marshal(map[string]string{aliyunVerifyCodeParam: vals[0]})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func normalizeAliyunPhone(phone string) string {
	phone = strings.TrimPrefix(phone, "+86")
	return strings.TrimPrefix(phone, "86")
}

func configStr(config map[string]interface{}, key string) string {
	v, ok := config[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
