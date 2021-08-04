package face

import (
	"fmt"
	"github.com/liuhengloveyou/passport/protos"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	. "github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
)

func modifyAvatarByForm(w http.ResponseWriter, r *http.Request) {
	var (
		dir string
		fp  string
	)

	var uid uint64
	if r.Context().Value("session") != nil {
		uid = r.Context().Value("session").(*sessions.Session).Values[SessUserInfoKey].(protos.User).UID
	}
	if uid <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		logger.Error("modifyAvatarByForm session ERR.")
		return
	}

	flen, _ := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)
	if flen == 0 || flen > MAX_UPLOAD_LEN {
		logger.Error("FileUpload Content-Length ERR: ", flen)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "文件大小错误")
		return

	}

	r.ParseMultipartForm(32 << 20)

	fileType := r.FormValue("type")
	if fileType == "" {
		logger.Error("FileUpload fileType nil")
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "文件类型错误")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		logger.Error("FileUpload FormFile err: ", err)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "读上传文件错误")
		return
	}
	defer file.Close()

	fileBuff, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Error("FileUpload ReadAll err: ", err)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "读上传文件错误")
		return
	}

	dir = fmt.Sprintf("%s/%d/%d", ServConfig.AvatarDir, time.Now().Year(), time.Now().Month())
	if err := os.MkdirAll(dir, 0755); err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, "文件系统错误")
		logger.Error("FileUpload mkdir ERR: ", dir, err)
		return
	}

	fp = fmt.Sprintf("%s/%d.%s", dir, uid, fileType)
	logger.Info("FileUpload fn: ", fp)

	if err := ioutil.WriteFile(fp, fileBuff, 0755); err != nil {
		logger.Error("FileUpload err: ", err)
		gocommon.HttpErr(w, http.StatusInternalServerError, -1, "写文件失败")
		return
	}

	logger.Info("FileUpload ok: ", fp)

	// 更新用户信息到数据库
	if _, err = service.UpdateUserService(&protos.UserReq{UID: uid, AvatarURL: fmt.Sprintf("%d/%d/%d.%s", time.Now().Year(), time.Now().Month(), uid, fileType)}); err != nil {
		logger.Errorf("modifyAvatarByForm service ERR:", err)
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, fmt.Sprintf("%d/%d/%d.%s", time.Now().Year(), time.Now().Month(), uid, fileType))

	return
}
