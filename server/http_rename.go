package server

import (
	"net/http"
	"os"
	"strings"
)

// RenameFileOrFolder 重命名文件或文件夹
// 请求参数1 oldPath  源(旧)文件或文件夹完整路径 (文件包含文件名)
// 请求参数2 newPath 目标(新)文件或文件夹完整路径 (文件包含文件名)
// 请求参数3 md5 用于重新生成并设置文件信息
func (c *Server) RenameFileOrFolder(w http.ResponseWriter, r *http.Request) {
	var (
		result JsonResult
	)
	result.Status = "ok"
	if c.IsPeer(r) {
		r.ParseForm()
		oldPath := r.FormValue("oldPath")
		newPath := r.FormValue("newPath")
		newPathBackup := newPath
		md5 := r.FormValue("md5")
		// 文件夹不传md5 所以为空
		if md5 == "" {
			md5s := c.getDirFilesMd5(strings.Replace(oldPath, "files/", "", 1))
			for _, s := range md5s {
				v, err := c.GetFileInfoFromLevelDB(s)
				newPath = strings.Replace(v.Path, oldPath, newPath, 1)
				md5 = c.util.MD5(newPath + "/" + v.Name)
				if err == nil {
					v.Md5 = md5
					v.Path = newPath
				}
				c.saveFileMd5Log(v, CONST_FILE_Md5_FILE_NAME)
			}
			os.Rename(oldPath, newPathBackup)
		} else {
			newFileName := string([]rune(newPath)[UnicodeIndex(newPath, "/")+1:])
			v, err := c.GetFileInfoFromLevelDB(md5)
			md5 = c.util.MD5(newPath)
			if err == nil {
				//fmt.Println(v.Name)
				if v.Name != "" {
					tmpPath := v.Path + "/" + v.Name
					// 不等
					if strings.Compare(tmpPath, oldPath) != 0 {
						v.Name = newFileName
					}
				}
				v.Name = newFileName
				v.Md5 = md5
			}
			if c.util.FileExists(oldPath) {
				os.Rename(oldPath, newPath)
				c.saveFileMd5Log(v, CONST_FILE_Md5_FILE_NAME)
			} else {
				result.Status = "error"
				return
			}
			result.Status = "ok"
		}
		result.Message = "rename job start ..,don't try again!!!"
		w.Write([]byte(c.util.JsonEncodePretty(result)))
	} else {
		result.Message = c.GetClusterNotPermitMessage(r)
		w.Write([]byte(c.util.JsonEncodePretty(result)))
	}

}

func UnicodeIndex(str, substr string) int {
	// 子串在字符串的字节位置
	result := strings.LastIndex(str, substr)
	if result >= 0 {
		// 获得子串之前的字符串并转换成[]byte
		prefix := []byte(str)[0:result]
		// 将子串之前的字符串转换成[]rune
		rs := []rune(string(prefix))
		// 获得子串之前的字符串的长度，便是子串在字符串的字符位置
		result = len(rs)
	}

	return result
}
