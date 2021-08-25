package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego/httplib"
	log "github.com/sjqzhang/seelog"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (c *Server) RemoveDownloading() {
	RemoveDownloadFunc := func() {
		for {
			iter := c.ldb.NewIterator(util.BytesPrefix([]byte("downloading_")), nil)
			for iter.Next() {
				key := iter.Key()
				keys := strings.Split(string(key), "_")
				if len(keys) == 3 {
					if t, err := strconv.ParseInt(keys[1], 10, 64); err == nil && time.Now().Unix()-t > 60*10 {
						os.Remove(DOCKER_DIR + keys[2])
					}
				}
			}
			iter.Release()
			time.Sleep(time.Minute * 3)
		}
	}
	go RemoveDownloadFunc()
}

// RemoveEmptyDir 移除空目录
func (c *Server) RemoveEmptyDir(w http.ResponseWriter, r *http.Request) {
	var (
		result JsonResult
	)
	result.Status = "ok"
	// 如果是集群
	if c.IsPeer(r) {
		// 删除data目录下空目录
		go c.util.RemoveEmptyDir(DATA_DIR)
		// 删除files目录下空目录
		go c.util.RemoveEmptyDir(STORE_DIR)
		result.Message = "clean job start ..,don't try again!!!"
		w.Write([]byte(c.util.JsonEncodePretty(result)))
	} else {
		result.Message = c.GetClusterNotPermitMessage(r)
		w.Write([]byte(c.util.JsonEncodePretty(result)))
	}
}

// RemoveDir 移除目录 包含文件
func (c *Server) RemoveDir(w http.ResponseWriter, r *http.Request) {
	var (
		_      *FileInfo
		result JsonResult
		path   string
	)
	result.Status = "ok"
	// 如果是集群
	if c.IsPeer(r) {
		r.ParseForm()
		path = r.FormValue("path")
		// 删除db记录的md5
		md5s := c.getDirFilesMd5(path)
		for _, m := range md5s {
			c.RemoveKeyFromLevelDB(m, c.ldb)
		}
		// 删除files目录下目录
		err := os.RemoveAll(STORE_DIR + "/" + path)
		if err != nil {
			result.Status = "error"
		}
		result.Message = "clean job start ..,don't try again!!!"
		w.Write([]byte(c.util.JsonEncodePretty(result)))
	} else {
		result.Message = c.GetClusterNotPermitMessage(r)
		w.Write([]byte(c.util.JsonEncodePretty(result)))
	}
}

// RemoveFile 移除文件
func (c *Server) RemoveFile(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		md5sum   string
		fileInfo *FileInfo
		fpath    string
		delUrl   string
		result   JsonResult
		inner    string
		name     string
	)
	_ = delUrl
	_ = inner
	r.ParseForm()
	md5sum = r.FormValue("md5")
	fpath = r.FormValue("path")
	inner = r.FormValue("inner")
	result.Status = "fail"
	if !c.IsPeer(r) {
		w.Write([]byte(c.GetClusterNotPermitMessage(r)))
		return
	}
	if Config().AuthUrl != "" && !c.CheckAuth(w, r) {
		c.NotPermit(w, r)
		return
	}
	// 路径不为空并且文件md5为空
	if fpath != "" && md5sum == "" {
		if Config().Group != "" && Config().SupportGroupManage {
			fpath = strings.Replace(fpath, "/"+Config().Group+"/", STORE_DIR_NAME+"/", 1)
			md5sum = c.util.MD5(STORE_DIR_NAME + "/" + fpath)
		} else {
			fpath = strings.Replace(fpath, "/", STORE_DIR_NAME+"/", 1)
		}

	}
	if inner != "1" {
		for _, peer := range Config().Peers {
			delFile := func(peer string, md5sum string, fileInfo *FileInfo) {
				delUrl = fmt.Sprintf("%s%s", peer, c.getRequestURI("delete"))
				req := httplib.Post(delUrl)
				req.Param("md5", md5sum)
				req.Param("inner", "1")
				req.SetTimeout(time.Second*5, time.Second*10)
				if _, err = req.String(); err != nil {
					log.Error(err)
				}
			}
			go delFile(peer, md5sum, fileInfo)
		}
	}
	// 强制删除
	if fpath != "" && md5sum != "" {
		err := os.RemoveAll(STORE_DIR_NAME + "/" + fpath)
		c.RemoveKeyFromLevelDB(md5sum, c.ldb)
		if err != nil {
			result.Message = "fail remove"
			w.Write([]byte(c.util.JsonEncodePretty(result)))
			return
		} else {
			result.Message = "remove success"
			result.Status = "ok"
			w.Write([]byte(c.util.JsonEncodePretty(result)))
			return
		}
	}

	// 以下是按照md5删除文件
	if len(md5sum) < 32 {
		result.Message = "md5 unvalid"
		w.Write([]byte(c.util.JsonEncodePretty(result)))
		return
	}

	if fileInfo, err = c.GetFileInfoFromLevelDB(md5sum); err != nil {
		result.Message = err.Error()
		w.Write([]byte(c.util.JsonEncodePretty(result)))
		return
	}
	if fileInfo.OffSet >= 0 {
		result.Message = "small file delete not support"
		w.Write([]byte(c.util.JsonEncodePretty(result)))
		return
	}
	name = fileInfo.Name
	if fileInfo.ReName != "" {
		name = fileInfo.ReName
	}

	// 删除该路径文件
	fpath = fileInfo.Path + "/" + name
	if fileInfo.Path != "" && c.util.FileExists(DOCKER_DIR+fpath) {
		fileInfo.Md5 = c.util.MD5(fpath)
		c.RemoveKeyFromLevelDB(fileInfo.Md5, c.ldb)
		if err = os.Remove(DOCKER_DIR + fpath); err != nil {
			result.Message = err.Error()
			w.Write([]byte(c.util.JsonEncodePretty(result)))
			return
		} else {
			result.Message = "remove success"
			result.Status = "ok"
			w.Write([]byte(c.util.JsonEncodePretty(result)))
			return
		}
	}
	result.Message = "fail remove"
	w.Write([]byte(c.util.JsonEncodePretty(result)))
}

// 根据文件夹路径获取其中所有文件的md5
func (c *Server) getDirFilesMd5(dir string) []string {
	var (
		filesInfo []os.FileInfo
		md5s      []string
		err       error
		tmpDir    string
	)
	dir = strings.Replace(dir, ".", "", -1)
	if tmpDir, err = os.Readlink(dir); err == nil {
		dir = tmpDir
	}
readDir:
	filesInfo, err = ioutil.ReadDir(DOCKER_DIR + STORE_DIR_NAME + "/" + dir)
	if err != nil {
		log.Error(err)
	}
	for _, f := range filesInfo {
		fi := FileInfoResult{
			Name:    f.Name(),
			Size:    f.Size(),
			IsDir:   f.IsDir(),
			ModTime: f.ModTime().Unix(),
			Path:    dir,
			Md5:     c.util.MD5(strings.Replace(STORE_DIR_NAME+"/"+dir+"/"+f.Name(), "//", "/", -1)),
		}
		if fi.IsDir {
			dir = dir + "/" + fi.Name
			goto readDir
		} else {
			md5s = append(md5s, fi.Md5)
		}
	}
	return md5s
}
