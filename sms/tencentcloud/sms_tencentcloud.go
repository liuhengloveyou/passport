package tencentcloud

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	txsms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111" // 引入sms

	sms "github.com/liuhengloveyou/passport/sms"
)

type SmsTencentcloud struct {
	SecretID  string
	SecretKey string

	UserAddTemplateId string // 注册用户验证码短信模板ID
}

func init() {
	sms.Register("tencentcloud", NewSmsTencentcloud)
}

func NewSmsTencentcloud(config map[string]interface{}) sms.Sms {
	tmp := &SmsTencentcloud{}
	tmp.SecretID = config["secretId"].(string)
	tmp.SecretKey = config["secretKey"].(string)
	tmp.UserAddTemplateId = config["userAddTemplateId"].(string)
	if len(tmp.SecretID) == 0 ||
		len(tmp.SecretKey) == 0 ||
		len(tmp.UserAddTemplateId) == 0 {
		return nil
	}

	return tmp
}

// 注册用户验证码
func (p *SmsTencentcloud) SendUserAddSms(phoneNumber string, aliveSecond int64) (code string, err error) {
	code = fmt.Sprintf("%06v", rand.New(rand.NewSource((time.Now().UnixNano()))).Int31n(1000000))

	if err = p.sendSms([]string{phoneNumber}, "", p.UserAddTemplateId, []string{code}); err != nil {
		return
	}

	return
}

func (p *SmsTencentcloud) sendSms(phoneNumber []string, userSession string, TemplateId string, TemplateVals []string) (err error) {
	/* 必要步骤：
	 * 实例化一个认证对象，入参需要传入腾讯云账户密钥对secretId，secretKey。
	 * 这里采用的是从环境变量读取的方式，需要在环境变量中先设置这两个值。
	 * 你也可以直接在代码中写死密钥对，但是小心不要将代码复制、上传或者分享给他人，
	 * 以免泄露密钥对危及你的财产安全。
	 * CAM密匙查询: https://console.cloud.tencent.com/cam/capi*/
	credential := common.NewCredential(p.SecretID, p.SecretKey)

	/* 非必要步骤:
	 * 实例化一个客户端配置对象，可以指定超时时间等配置 */
	cpf := profile.NewClientProfile()

	/* SDK默认使用POST方法。
	 * 如果你一定要使用GET方法，可以在这里设置。GET方法无法处理一些较大的请求 */
	cpf.HttpProfile.ReqMethod = "POST"

	/* SDK有默认的超时时间，非必要请不要进行调整
	 * 如有需要请在代码中查阅以获取最新的默认值 */
	// cpf.HttpProfile.ReqTimeout = 5

	/* SDK会自动指定域名。通常是不需要特地指定域名的，但是如果你访问的是金融区的服务
	 * 则必须手动指定域名，例如sms的上海金融区域名： sms.ap-shanghai-fsi.tencentcloudapi.com */
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"

	/* SDK默认用TC3-HMAC-SHA256进行签名，非必要请不要修改这个字段 */
	cpf.SignMethod = "HmacSHA1"

	/* 实例化要请求产品(以sms为例)的client对象
	 * 第二个参数是地域信息，可以直接填写字符串ap-guangzhou，或者引用预设的常量 */
	client, err := txsms.NewClient(credential, "ap-guangzhou", cpf)
	if err != nil {
		fmt.Println("sms.NewClient ERR: ", err)
		return err
	}

	/* 实例化一个请求对象，根据调用的接口和实际情况，可以进一步设置请求参数
	 * 你可以直接查询SDK源码确定接口有哪些属性可以设置
	 * 属性可能是基本类型，也可能引用了另一个数据结构
	 * 推荐使用IDE进行开发，可以方便的跳转查阅各个接口和数据结构的文档说明 */
	request := txsms.NewSendSmsRequest()

	/* 基本类型的设置:
	 * SDK采用的是指针风格指定参数，即使对于基本类型你也需要用指针来对参数赋值。
	 * SDK提供对基本类型的指针引用封装函数
	 * 帮助链接：
	 * 短信控制台: https://console.cloud.tencent.com/smsv2
	 * sms helper: https://cloud.tencent.com/document/product/382/3773 */

	/* 短信应用ID: 短信SdkAppId在 [短信控制台] 添加应用后生成的实际SdkAppId，示例如1400006666 */
	// https://console.cloud.tencent.com/smsv2/app-manage
	// 系统默认应用：1400591645
	request.SmsSdkAppId = common.StringPtr("1400591645")
	/* 短信签名内容: 使用 UTF-8 编码，必须填写已审核通过的签名，签名信息可登录 [短信控制台] 查看 */
	request.SignName = common.StringPtr("")
	/* 国际/港澳台短信 SenderId: 国内短信填空，默认未开通，如需开通请联系 [sms helper] */
	request.SenderId = common.StringPtr("")
	/* 用户的 session 内容: 可以携带用户侧 ID 等上下文信息，server 会原样返回 */
	request.SessionContext = common.StringPtr(userSession)
	/* 短信码号扩展号: 默认未开通，如需开通请联系 [sms helper] */
	request.ExtendCode = common.StringPtr("")
	/* 模板参数: 若无模板参数，则设置为空*/
	request.TemplateParamSet = common.StringPtrs(TemplateVals)
	/* 模板 ID: 必须填写已审核通过的模板 ID。模板ID可登录 [短信控制台] 查看 */
	request.TemplateId = common.StringPtr(TemplateId)
	/* 下发手机号码，采用 E.164 标准，+[国家或地区码][手机号]
	 * 示例如：+8613711112222， 其中前面有一个+号 ，86为国家码，13711112222为手机号，最多不要超过200个手机号*/
	request.PhoneNumberSet = common.StringPtrs(phoneNumber)

	// 通过client对象调用想要访问的接口，需要传入请求对象
	response, err := client.SendSms(request)
	// 处理异常
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
	}
	// 非SDK异常，直接失败。实际代码中可以加入其他的处理。
	if err != nil {
		panic(err)
	}

	// 打印返回的json字符串
	json.Marshal(response.Response)
	// fmt.Printf("%v %v %v %v>>>>>>>>>%s\n", phoneNumber[0], userSession, TemplateId, TemplateVals, b)
	if *response.Response.SendStatusSet[0].Fee < 1 {
		return fmt.Errorf("%s", *response.Response.SendStatusSet[0].Code)
	}

	return nil
}
