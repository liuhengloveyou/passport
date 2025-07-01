package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/sms"
	"go.uber.org/zap"

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
		if duplicatePhone(p.Cellphone) {
			return 0, common.ErrPhoneDup
		}
		if p.Nickname == "" {
			p.Nickname = p.Cellphone
		}
	}

	if p.Email != "" {
		if duplicateEmail(p.Email) {
			return 0, common.ErrEmailDup
		}
		if p.Nickname == "" {
			p.Nickname = p.Email
		}
	}

	if p.Nickname != "" {
		if duplicateNickname(p.Nickname) {
			return 0, common.ErrNickDup
		}
	}

	if len(p.WxOpenId) > 0 {
		if duplicateWxOpenid(p.WxOpenId) {
			return 0, common.ErrWxOpenidDup
		}
	}

	p.Password = common.EncryPWD(p.Password)

	uid, err := dao.UserInsert(p)
	if err != nil {
		common.Logger.Sugar().Errorf("dao.UserInsert ERR: %v\n", err)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" { // 唯一约束冲突
			return 0, common.ErrPgDupKey
		}
		return 0, common.ErrService
	}

	return uint64(uid), err
}

func GetUserInfo(uid uint64) (r *protos.User, e error) {
	if uid <= 0 {
		return nil, common.ErrParam
	}

	r, e = dao.UserQueryByID(uid)
	if e != nil {
		return nil, e
	}
	if r == nil {
		return nil, common.ErrUserNotFound
	}
	r.Password = ""

	return
}

func AdminUserList(tenantID, page, pageSize uint64, hasTotal bool, nickname string) (rst protos.PageResponse, e error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	var rr []protos.User
	var uids []uint64
	rr, e = dao.UserQueryByTenant(tenantID, page, pageSize, nickname, uids)
	if e != nil {
		common.Logger.Sugar().Error("AdminUserList db ERR: ", e)
		e = common.ErrService
		return
	}
	if len(rr) == 0 {
		e = common.ErrNull
		return
	}
	rst.List = rr

	if hasTotal {
		rst.Total, e = dao.UserCountByTenant(tenantID, nickname, uids)
		if e != nil {
			common.Logger.Sugar().Error("AdminUserList count ERR: ", e)
			e = common.ErrService
			return
		}
	}

	return
}

func SelectUsersLite(m *protos.UserReq) (rr []protos.UserLite, e error) {
	if m.TenantID <= 0 {
		return nil, common.ErrParam
	}

	rr, e = dao.UserLiteQuery(m, m.PageNo, m.PageSize)

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
		if duplicatePhone(p.Cellphone) {
			return -1, fmt.Errorf("电话号码重复")
		}
	}

	if p.Email != "" {
		if duplicateEmail(p.Email) {
			return -1, fmt.Errorf("邮箱重复")
		}
	}

	if p.Nickname != "" {
		if duplicateNickname(p.Nickname) {
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

func UpdateUserPWDBySms(cellphone, smsCode, newPWD string) (rows int64, e error) {
	if len(cellphone) == 0 || len(smsCode) == 0 || len(newPWD) == 0 {
		return 0, common.ErrParam
	}

	e = sms.CheckSmsCode(cellphone, smsCode)
	if e != nil {
		return -1, e
	}

	newPWD = common.EncryPWD(newPWD)

	rows, e = dao.UserUpdatePWDByCellphone(cellphone, newPWD)
	if rows < 1 || e != nil {
		common.Logger.Error("UpdateUserPWDBySms ERR: ", zap.Any("row", rows), zap.Error(e))
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

func GetUserInfoService(uid, tenantId uint64) (r *protos.User, e error) {
	if uid <= 0 {
		e = fmt.Errorf("uid nil")
		return
	}
	model := &protos.UserReq{
		UID: uid,
	}

	if r, e = dao.UserQueryOne(model); e != nil {
		common.Logger.Sugar().Errorf("GetUserInfoService DB ERR: %v\n", e)
		return
	}
	if r == nil {
		common.Logger.Sugar().Errorf("GetUserInfoService len ERR: %v %v\n", r, e)
		return
	}

	r.Password = ""

	if tenantId > 0 {
		if r.Roles, e = getTenantUserRoles(uid, tenantId); e != nil {
			common.Logger.Sugar().Warnf("GetUserInfoService getTenantUserRole ERR: %v", e)
		}

		// 部门字典
		var depIds []uint64
		if deps, ok := r.Ext["deps"].([]interface{}); ok {
			for _, dep := range deps {
				depIds = append(depIds, uint64(dep.(float64)))
			}
		}
		if r.Departments, e = getTenantUserDepartment(uid, tenantId, depIds); e != nil {
			common.Logger.Sugar().Warnf("GetUserInfoService getTenantUserDepartment ERR: %v", e)
			e = nil
		}
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

	e = dao.BusinessQuery(model, models, 1, 1)

	return
}

func duplicatePhone(phone string) (has bool) {
	rr, err := dao.UserQuery(&protos.UserReq{Cellphone: phone}, 1, 1)
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
	rr, err := dao.UserQuery(&protos.UserReq{Email: email}, 1, 1)
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
	rr, err := dao.UserQuery(&protos.UserReq{Nickname: nickname}, 1, 100)
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

func duplicateWxOpenid(openid string) (has bool) {
	rr, err := dao.UserQuery(&protos.UserReq{WxOpenId: openid}, 1, 100)
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

	p.SmsCode = strings.TrimSpace(p.SmsCode)

	if e := common.Validate.Struct(p); e != nil {
		return errors.New(e.(validator.ValidationErrors)[0].Translate(common.Trans))
	}

	return nil
}

func UpdateUserWxOpenIdByCellphone(cellphone, wxOpenId, smsCode string) (rows int64, e error) {
	if len(cellphone) == 0 || len(wxOpenId) == 0 {
		return 0, common.ErrParam
	}

	e = sms.CheckSmsCode(cellphone, smsCode)
	if e != nil {
		return -1, e
	}

	rows, e = dao.UserUpdateWxOpenIdByCellphone(cellphone, wxOpenId)
	if e != nil || rows < 1 {
		common.Logger.Error("UpdateUserWxOpenIdByCellphone ERR: ", zap.String("cellphone", cellphone), zap.String("wxOpenId", wxOpenId), zap.Any("row", rows), zap.Error(e))
		return 0, common.ErrModify
	}

	return
}
