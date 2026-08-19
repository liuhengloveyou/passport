package user

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/face/core"
	"github.com/liuhengloveyou/passport/v4/protos"
	"github.com/liuhengloveyou/passport/v4/service"
)

const avatarExt = "png"

// UserModifyAvatarForm 通过 multipart/form-data 上传并更新用户头像。
func UserModifyAvatarForm(w http.ResponseWriter, r *http.Request) {
	uid := core.GetSessionUser(r).UID
	if uid <= 0 {
		common.Logger.Sugar().Errorf("user.UserModifyAvatarForm no auth: method=%s uri=%s", r.Method, r.RequestURI)
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		return
	}

	if err := r.ParseMultipartForm(common.MAX_UPLOAD_LEN); err != nil {
		common.Logger.Sugar().Errorf("user.UserModifyAvatarForm ParseMultipartForm failed: uid=%d err=%v", uid, err)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "文件大小错误")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		common.Logger.Sugar().Errorf("user.UserModifyAvatarForm FormFile failed: uid=%d err=%v", uid, err)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "读上传文件错误")
		return
	}
	defer file.Close()

	if header.Size > 0 && header.Size > common.MAX_UPLOAD_LEN {
		common.Logger.Sugar().Errorf("user.UserModifyAvatarForm bad file size: uid=%d size=%d max=%d", uid, header.Size, common.MAX_UPLOAD_LEN)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "文件大小错误")
		return
	}

	common.Logger.Sugar().Infof("user.UserModifyAvatarForm start: uid=%d size=%d", uid, header.Size)

	dir := common.ServConfig.AvatarDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		common.Logger.Sugar().Errorf("user.UserModifyAvatarForm MkdirAll failed: uid=%d dir=%s err=%v", uid, dir, err)
		gocommon.HttpErr(w, http.StatusOK, -1, "文件系统错误")
		return
	}

	filename := fmt.Sprintf("%d.%s", uid, avatarExt)
	fp := filepath.Join(dir, filename)
	out, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		common.Logger.Sugar().Errorf("user.UserModifyAvatarForm OpenFile failed: uid=%d path=%s err=%v", uid, fp, err)
		gocommon.HttpErr(w, http.StatusInternalServerError, -1, "写文件失败")
		return
	}
	defer out.Close()

	written, err := io.Copy(out, io.LimitReader(file, common.MAX_UPLOAD_LEN+1))
	if err != nil {
		common.Logger.Sugar().Errorf("user.UserModifyAvatarForm Copy failed: uid=%d path=%s err=%v", uid, fp, err)
		gocommon.HttpErr(w, http.StatusInternalServerError, -1, "写文件失败")
		return
	}
	if written == 0 || written > common.MAX_UPLOAD_LEN {
		common.Logger.Sugar().Errorf("user.UserModifyAvatarForm bad file size: uid=%d written=%d max=%d", uid, written, common.MAX_UPLOAD_LEN)
		_ = os.Remove(fp)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "文件大小错误")
		return
	}

	avatarURL := fmt.Sprintf("avatar/%s", filename)
	if _, err = service.UpdateUserService(&protos.UserReq{UID: uid, AvatarURL: avatarURL}); err != nil {
		common.Logger.Sugar().Errorf("user.UserModifyAvatarForm UpdateUserService failed: uid=%d avatar=%s err=%v", uid, avatarURL, err)
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}

	common.Logger.Sugar().Infof("user.UserModifyAvatarForm success: uid=%d avatar=%s size=%d", uid, avatarURL, written)
	gocommon.HttpErr(w, http.StatusOK, 0, filename)
}
