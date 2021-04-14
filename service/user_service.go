package service

import (
	"errors"
	"fmt"
	"github.com/liuhengloveyou/passport/protos"
	"strings"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"

	validator "github.com/go-playground/validator/v10"
)

func AddUserService(p *protos.UserReq) (id int64, e error) {
	if p.Cellphone == "" && p.Email == "" {
		return -1, fmt.Errorf("手机号和邮箱同时为空")
	}
	if p.Password == "" {
		return -1, fmt.Errorf("用户密码为空")
	}

	if e = userPreTreat(p); e != nil {
		return -1, e
	}

	if p.Cellphone != "" {
		if duplicatePhone(p.Cellphone) == true {
			return -1, fmt.Errorf("手机号码重复")
		}
		if p.Nickname == "" {
			p.Nickname = p.Cellphone
		}
	}

	if p.Email != "" {
		if duplicateEmail(p.Email) == true {
			return -1, fmt.Errorf("邮箱重复")
		}
		if p.Nickname == "" {
			p.Nickname = p.Email
		}
	}

	if p.Nickname != "" {
		if duplicateNickname(p.Nickname) == true {
			return -1, fmt.Errorf("昵称重复")
		}
	}

	p.Password = common.EncryPWD(p.Password)

	return dao.UserInsert(p)
}


func GetUser(m *protos.UserReq) (r *protos.User, e error) {
	if m.UID > 0 {
		r, e = dao.UserSelectByID(m.UID)
	}

	return
}

func SelectUsers(m *protos.UserReq) (rr []protos.User, e error) {
	if m.TenantID > 0 {
		rr, e = dao.UserSelectByTenantID(m.TenantID)
	}

	return
}

func UpdateUserService(p *protos.UserReq) (rows int64, e error) {
	if p.UID <= 0 {
		return 0, fmt.Errorf("用户错误")
	}

	if e = userPreTreat(p); e != nil {
		return -1, e
	}

	if e := common.Validate.Struct(p); e != nil {
		validationErrors := e.(validator.ValidationErrors)
		for _, validationErr := range validationErrors {
			if validationErr.ActualTag() != "required" {
				return 0, fmt.Errorf("数据格式有误")
			}
		}
	}

	if p.Cellphone != "" {
		if duplicatePhone(p.Cellphone) == true {
			return -1, fmt.Errorf("电话号码重复")
		}
	}

	if p.Email != "" {
		if duplicateEmail(p.Email) == true {
			return -1, fmt.Errorf("邮箱重复")
		}
	}

	if p.Nickname != "" {
		if duplicateNickname(p.Nickname) == true {
			return -1, fmt.Errorf("昵称重复")
		}
	}

	return dao.UserUpdate(p)
}

func UpdateUserPWD(uid uint64, oldPWD, newPWD string) (rows int64, e error) {
	if uid <= 0 {
		return 0, fmt.Errorf("用户错误")
	}

	if oldPWD == "" {
		return -1, fmt.Errorf("旧密码不能为空")
	} else if newPWD == "" {
		return -1, fmt.Errorf("新密码不能为空")
	}

	newPWD = common.EncryPWD(newPWD)
	oldPWD = common.EncryPWD(oldPWD)

	rows, e = dao.UserUpdatePWD(uid, oldPWD, newPWD)
	if rows < 1 {
		return 0, fmt.Errorf("更改密码失败")
	}

	return
}

func duplicatePhone(phone string) (has bool) {
	rr, err := dao.UserSelect(&protos.UserReq{Cellphone: phone}, 1, 1)
	if err != nil {
		return false
	}
	if len(rr) == 0 {
		return false
	}
	if rr[0].UID > 0 {
		return true
	}

	return false
}

func duplicateEmail(email string) (has bool) {
	rr, err := dao.UserSelect(&protos.UserReq{Email: email}, 1, 1)
	if err != nil {
		return false
	}
	if len(rr) == 0 {
		return false
	}
	if rr[0].UID > 0 {
		return true
	}

	return false
}

func duplicateNickname(nickname string) (has bool) {
	rr, err := dao.UserSelect(&protos.UserReq{Nickname: nickname}, 1, 100)
	if err != nil {
		return false
	}
	if len(rr) == 0 {
		return false
	}
	if rr[0].UID <= 0 {
		return true
	}

	return false

}

// 格式预处理
func userPreTreat(p *protos.UserReq) error {
	if p.Cellphone != "" {
		p.Cellphone = strings.TrimSpace(strings.ToLower(p.Cellphone))
	}

	if p.Email != "" {
		p.Email = strings.TrimSpace(strings.ToLower(p.Email))
	}

	if p.Nickname != "" {
		p.Nickname = strings.TrimSpace(strings.TrimSpace(p.Nickname))
	}
	if p.AvatarURL != "" {
		if strings.HasPrefix(p.AvatarURL, ".") {
			p.AvatarURL = p.AvatarURL[1:]
		}
	}

	if e := common.Validate.Struct(p); e != nil {
		return errors.New(e.(validator.ValidationErrors)[0].Translate(common.Trans))
	}

	return nil
}
