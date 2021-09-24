package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/astaxie/beego/httplib"
	log "github.com/sjqzhang/seelog"
)

func (c *Server) CheckAuth(w http.ResponseWriter, r *http.Request) bool {
	var (
		err        error
		req        *httplib.BeegoHTTPRequest
		result     string
		token      string
		jsonResult JsonResult
	)
	if err = r.ParseForm(); err != nil {
		log.Error(err)
		return false
	}
	println("post 验证token")
	if Config().DownloadUseToken && strings.Contains(r.RequestURI, "auth_token=") {
		tmpUrl := r.RequestURI
		begin := strings.LastIndex(tmpUrl, "auth_token=") + 11
		token = tmpUrl[begin : begin+32]
		group := Config().Group
		path := tmpUrl[strings.Index(tmpUrl, group+"/")+len(group)+1 : strings.Index(tmpUrl, "auth_token")-1]
		if token == "" {
			return false
		}
		//path = url.QueryEscape(path)
		token = strings.TrimSpace(token)
		println("token:", token)
		println("token len:", len(token))
		println("path:", path)
		req = httplib.Post(Config().AuthUrl).Param("auth_token", token).Param("path", path)
		req.SetTimeout(time.Second*10, time.Second*10)
		req.Param("__path__", r.URL.Path)
		req.Param("__query__", r.URL.RawQuery)
		for k, _ := range r.Form {
			req.Param(k, r.FormValue(k))
		}
		for k, v := range r.Header {
			req.Header(k, v[0])
		}
		result, err = req.String()
		result = strings.TrimSpace(result)
		if strings.HasPrefix(result, "{") && strings.HasSuffix(result, "}") {
			if err = json.Unmarshal([]byte(result), &jsonResult); err != nil {
				log.Error(err)
				return false
			}
			println("57:", jsonResult.Data)
			if jsonResult.Data != "ok" {
				log.Warn(result)
				return false
			}
		} else {
			println("63", result)
			if result != "ok" {
				log.Warn(result)
				return false
			}
		}
	}
	return true
}

func (c *Server) NotPermit(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(401)
}
