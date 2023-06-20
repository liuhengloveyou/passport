package tencentcloud

import "testing"

// go test -v -run ^TestSendUserAddSms$ github.com/liuhengloveyou/passport/sms/tencentcloud
func TestSendUserAddSms(t *testing.T) {
	sms := NewSmsTencentcloud(map[string]interface{}{
		"secret_id":            "xxx",
		"secret_key":           "xxx",
		"sdk_app_id":           "xxx",
		"sign_name":            "xxx",
		"user_add_template_id": "xxx",
	})

	sms.SendUserAddSms("17612116527", 30)
}
