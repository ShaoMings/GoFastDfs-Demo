package server

import (
	log "github.com/sjqzhang/seelog"
	"net/http"
	"os"
)

func (c *Server) CreateFolder(w http.ResponseWriter, r *http.Request) {
	var (
		result JsonResult
		err    error
	)
	result.Status = "ok"
	// 如果是集群
	if c.IsPeer(r) {
		r.ParseForm()
		// 获取创建的路径(包含新建的文件夹名)
		path := r.FormValue("path")
		// 文件夹不存在
		if !c.util.FileExists(path) {
			if err = os.MkdirAll(path, 0777); err != nil {
				log.Error(err)
			}
		} else {
			result.Status = "error"
		}
		result.Message = "create job start ..,don't try again!!!"
		w.Write([]byte(c.util.JsonEncodePretty(result)))
	} else {
		result.Message = c.GetClusterNotPermitMessage(r)
		w.Write([]byte(c.util.JsonEncodePretty(result)))
	}
}
