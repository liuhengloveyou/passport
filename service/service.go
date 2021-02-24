package service

import (
	"encoding/gob"
	"regexp"

	zhongwen "github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

var (
	validate *validator.Validate
	trans    ut.Translator
)

func init() {
	gob.Register(MiniAppUserInfo{})

	//
	validate = validator.New()
	validate.RegisterValidation("phone", CellPhoneValidate)

	zh := zhongwen.New()
	trans, _ = ut.New(zh, zh).GetTranslator("zh")

	zh_translations.RegisterDefaultTranslations(validate, trans)

	if err := validate.RegisterTranslation("phone", trans,
		func(ut ut.Translator) (err error) {
			return ut.Add("phone", "手机号码格式错误", true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, err := ut.T(fe.Tag(), fe.Field())
			if err != nil {
				return fe.(error).Error()
			}

			return t
		}); err != nil {
		panic(err)
	}

}

func CellPhoneValidate(fl validator.FieldLevel) bool {
	if fl.Field().Type().String() != "string" {
		return false
	}

	re, _ := regexp.Compile(`^1([378][0-9]|14[57]|5[^4])\d{8}$`)

	return re.MatchString(fl.Field().String())
}
