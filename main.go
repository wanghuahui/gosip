package main

import (
	_ "net/http/pprof"

	"github.com/panjjo/gosip/sip"
	"github.com/robfig/cron"
)

var (
	// sip服务用户信息
	_serverDevices NVRDevices
	srv            *sip.Server
)

func main() {

	srv = sip.NewServer()
	srv.RegistHandler(sip.REGISTER, handlerRegister)
	srv.RegistHandler(sip.MESSAGE, handlerMessage)
	go srv.ListenUDPServer("0.0.0.0:5060")
	RestfulAPI()
}

func init() {
	logger = newLogger()
	loadConfig()
	dbClient = NewClient().SetDB(config.DB.DBName)
	loadSYSInfo()
	streamCron()
}

// streamCron 定时任务，检查并关闭无效视频流
func streamCron() {
	c := cron.New()                         // 新建一个定时任务对象
	c.AddFunc("0 */5 * * * *", checkStream) // 定时关闭推送流,每5分钟执行一次
	c.Start()
}
