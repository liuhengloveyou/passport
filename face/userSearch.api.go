package face

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"

	gocommon "github.com/liuhengloveyou/go-common"
)

func searchLite(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	tenant, _ := strconv.ParseInt(r.FormValue("t"), 10, 64)
	if tenant <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	k := strings.TrimSpace(r.FormValue("k"))
	if k == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("pageSize"), 10, 64)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	userInfos, err := service.SelectUsersLite(&protos.UserReq{
		TenantID:  uint64(tenant),
		Nickname:  k,
		Cellphone: k,
		PageNo:    page,
		PageSize:  pageSize,
	})
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		logger.Error("searchLite ERR: " + err.Error())
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, userInfos)
	logger.Infof("searchLite OK: %#v %#v\n", k, userInfos)
}
