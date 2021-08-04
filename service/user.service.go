package service

import (
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/liuhengloveyou/passport/protos"
	"strings"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"

	validator "github.com/go-playground/validator/v10"
)

func AddUserService(p *protos.UserReq) (id uint64, e error) {
	if p.Cellphone == "" && p.Email == "" && p.Nickname == "" {
		return 0, common.ErrUserNmae
	}
	if p.Password == "" {
		return 0, common.ErrPWDNil
	}

	if e = userPreTreat(p); e != nil {
		common.Logger.Sugar().Errorf("AddUserService userPreTreat ERR: %v\n", e)
		return 0, common.ErrParam
	}

	if p.Cellphone != "" {
		if duplicatePhone(p.Cellphone) == true {
			return 0, common.ErrPhoneDup
		}
		if p.Nickname == "" {
			p.Nickname = p.Cellphone
		}
	}

	if p.Email != "" {
		if duplicateEmail(p.Email) == true {
			return 0, common.ErrEmailDup
		}
		if p.Nickname == "" {
			p.Nickname = p.Email
		}
	}

	if p.Nickname != "" {
		if duplicateNickname(p.Nickname) == true {
			return 0, common.ErrNickDup
		}
	}

	p.Password = common.EncryPWD(p.Password)

	uid, err := dao.UserInsert(p)
	if err != nil {
		common.Logger.Sugar().Errorf("dao.UserInsert ERR: %v\n", err)
		merr, ok := err.(*mysql.MySQLError)
		if ok && merr.Number == 1062 {
			return 0, common.ErrMysql1062
		}
		return 0, common.ErrService
	}
	
	return uint64(uid), err
}

func GetUser(m *protos.UserReq) (r *protos.User, e error) {
	if m.UID > 0 {
		r, e = dao.UserSelectByID(m.UID)
	}

	return
}

//func SelectUsers(m *protos.UserReq) (rr []protos.User, e error) {
//	if m.TenantID > 0 {
//		rr, e = dao.UserSelectByTenantID(m.TenantID, "")
//	}
//
//	return
//}

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
		return 0, common.ErrSession
	}

	if oldPWD == "" || newPWD == "" {
		return -1, common.ErrParam
	}

	newPWD = common.EncryPWD(newPWD)
	oldPWD = common.EncryPWD(oldPWD)

	rows, e = dao.UserUpdatePWD(uid, oldPWD, newPWD)
	if rows < 1 {
		return 0, common.ErrModify
	}

	return
}

func SetUserPWD(uid, tenantId uint64, PWD string) (rows int64, e error) {
	if uid <= 0 {
		return 0, common.ErrParam
	}
	if PWD == "" {
		return -1, common.ErrPWD
	}

	PWD = common.EncryPWD(PWD)

	rows, e = dao.SetUserPWD(uid, tenantId, PWD)
	if rows < 1 {
		common.Logger.Sugar().Errorf("SetUserPWD ERR: %d %d %v\n", uid, rows, e)
		return 0, common.ErrModify
	}

	return
}


func GetUserInfoService(uid uint64) (r protos.User, e error) {
	if uid <= 0 {
		e = fmt.Errorf("uid nil")
		return
	}

	model := &protos.UserReq{
		UID: uid,
	}

	var rr []protos.User
	if rr, e = dao.UserSelect(model, 1, 1); e != nil {
		common.Logger.Sugar().Errorf("GetUserInfoService DB ERR: %v\n", e)
		return
	}

	if rr != nil && len(rr) == 1 {
		rr[0].Password = ""
		rr[0].UpdateTime = nil
		return rr[0], nil
	}

	return
}

func GetBusinessUserInfoService(uid uint64, models interface{}) (e error) {
	if uid <= 0 {
		e = fmt.Errorf("uid nil")
		return
	}

	model := &protos.UserReq{
		UID: uid,
	}

	e = dao.BusinessSelect(model, models, 1, 1)

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
