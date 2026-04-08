package user

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
)

// UserSearchLite 按关键词在租户内分页搜索轻量用户列表。
func UserSearchLite(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenant, _ := strconv.ParseInt(r.FormValue("t"), 10, 64)
	k := strings.TrimSpace(r.FormValue("k"))
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("pageSize"), 10, 64)
	if tenant <= 0 || k == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}
	userInfos, err := service.SelectUsersLite(&protos.UserReq{TenantID: uint64(tenant), Nickname: k, Cellphone: k, PageNo: page, PageSize: pageSize})
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, userInfos)
}
