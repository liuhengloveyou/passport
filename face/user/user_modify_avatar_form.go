package user

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
	"github.com/liuhengloveyou/passport/v3/sessions"
)

// UserModifyAvatarForm 通过 multipart/form-data 上传并更新用户头像。
func UserModifyAvatarForm(w http.ResponseWriter, r *http.Request) {
	var uid uint64
	if r.Context().Value("session") != nil {
		uid = r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User).UID
	}
	if uid <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		return
	}
	flen, _ := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)
	if flen == 0 || flen > common.MAX_UPLOAD_LEN {
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "文件大小错误")
		return
	}
	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("file")
	if err != nil {
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "读上传文件错误")
		return
	}
	defer file.Close()
	fileBuff, err := io.ReadAll(file)
	if err != nil {
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "读上传文件错误")
		return
	}
	dir := fmt.Sprintf("%s/", common.ServConfig.AvatarDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, "文件系统错误")
		return
	}
	fp := fmt.Sprintf("%s/%d.%s", dir, uid, "png")
	if err := os.WriteFile(fp, fileBuff, 0755); err != nil {
		gocommon.HttpErr(w, http.StatusInternalServerError, -1, "写文件失败")
		return
	}
	if _, err = service.UpdateUserService(&protos.UserReq{UID: uid, AvatarURL: fmt.Sprintf("avatar/%d.%s", uid, "png")}); err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, fmt.Sprintf("%d%s", uid, "png"))
}
