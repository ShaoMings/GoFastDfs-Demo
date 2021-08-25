package main

import (
	dfs "github.com/sjqzhang/go-fastdfs/server"
	server "github.com/sjqzhang/go-fastdfs/server"
	_ "go.uber.org/automaxprocs" // 根据容器配额设置 maxprocs
	_ "net/http/pprof"           // 注册 pprof 接口
)

var (
	VERSION     string
	BUILD_TIME  string
	GO_VERSION  string
	GIT_VERSION string
)

func main() {
	dfs.VERSION = VERSION
	dfs.BUILD_TIME = BUILD_TIME
	dfs.GO_VERSION = GO_VERSION
	dfs.GIT_VERSION = GIT_VERSION
	//root := cobra.Command{Use: "fileserver"}
	//root.AddCommand(
	//	version.Cmd,
	//	doc.Cmd,
	//	server.Cmd,
	//)
	//root.Execute()
	server.InitServer()
	server.Start()
}
