package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/panjjo/gosip/sip"
	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	// ModDebug ModDebug
	ModDebug = "DEBUG"
	// SysInfoTbl sysinfo table
	SysInfoTbl = "sysinfo"
	// FileTbl file table
	FileTbl = "files"
)

type sysInfo struct {
	// LID 当前服务id
	LID string `json:"lid" bson:"lid" yaml:"lid" mapstructure:"lid"`
	// Region 当前域
	Region string `json:"region" bson:"region" yaml:"region" mapstructure:"region"`
	// UID 用户id固定头部
	UID string `json:"uid" bson:"uid"  yaml:"uid" mapstructure:"uid"`
	// UNUM 当前用户数
	UNUM int `json:"unum" bson:"unum" yaml:"unum" mapstructure:"unum"`
	// DID 设备id固定头部
	DID string `json:"did" bson:"did" yaml:"did" mapstructure:"did"`
	// DNUM 当前设备数
	DNUM int `json:"dnum" bson:"dnum" yaml:"dnum" mapstructure:"dnum"`
	// MediaServer 媒体服务
	MediaServer bool
	// 媒体服务器接流地址
	mediaServerRtpIP net.IP
	// 媒体服务器接流端口
	mediaServerRtpPort int
}

// ActiveDevices 记录当前活跃设备，请求播放时设备必须处于活跃状态
type ActiveDevices struct {
	sync.Map
}

var (
	// 活跃设备记录
	_activeDevices ActiveDevices
	// 系统运行信息
	_sysinfo sysInfo
)

func defaultInfo() sysInfo {
	return config.GB28181
}

// Get Get
func (a *ActiveDevices) Get(key string) (NVRDevices, bool) {
	if v, ok := a.Load(key); ok {
		return v.(NVRDevices), ok
	}
	return NVRDevices{}, false
}

func loadSYSInfo() {

	_activeDevices = ActiveDevices{sync.Map{}}

	_playList = playList{&sync.Map{}, &sync.Map{}, 0}
	ssrcLock = &sync.Mutex{}
	_recordList = &sync.Map{}

	// init sysinfo
	_sysinfo = sysInfo{}
	if err := dbClient.Get(SysInfoTbl, M{}, &_sysinfo); err != nil {
		if err == mongo.ErrNoDocuments {
			//  初始不存在
			_sysinfo = defaultInfo()
			if err = dbClient.Insert(SysInfoTbl, _sysinfo); err != nil {
				logrus.Fatalf("1 init sysinfo err:%v", err)
			}
		} else {
			logrus.Fatalf("2 init sysinfo err:%v", err)
		}
	}
	uri, _ := sip.ParseSipURI(fmt.Sprintf("sip:%s@%s", _sysinfo.LID, _sysinfo.Region))
	// uri, _ := sip.ParseSipURI(fmt.Sprintf("sip:%s@%s", _sysinfo.LID, config.UDP))
	_serverDevices = NVRDevices{
		DeviceID: _sysinfo.LID,
		Region:   _sysinfo.Region,
		addr: &sip.Address{
			DisplayName: sip.String{Str: "sipserver"},
			URI:         &uri,
			Params:      sip.NewParams(),
		},
	}

	// init media
	url, err := url.Parse(config.Media.RTP)
	if err != nil {
		logrus.Fatalf("media rtp url error,url:%s,err:%v", config.Media.RTP, err)
	}
	ipaddr, err := net.ResolveIPAddr("ip", url.Hostname())
	if err != nil {
		logrus.Fatalf("media rtp url error,url:%s,err:%v", config.Media.RTP, err)
	}
	_sysinfo.mediaServerRtpIP = ipaddr.IP
	_sysinfo.mediaServerRtpPort, _ = strconv.Atoi(url.Port())
}

// zlm接收到的ssrc为16进制。发起请求的ssrc为10进制
func ssrc2stream(ssrc string) string {
	if ssrc[0:1] == "0" {
		ssrc = ssrc[1:]
	}
	num, _ := strconv.Atoi(ssrc)
	return fmt.Sprintf("%X", num)
}

func sipResponse(tx *sip.Transaction) (*sip.Response, error) {
	response := tx.GetResponse()
	if response == nil {
		return nil, utils.NewError(nil, "response timeout", "tx key:", tx.Key())
	}
	if response.StatusCode() != http.StatusOK {
		return response, utils.NewError(nil, "response fail", response.StatusCode(), response.Reason(), "tx key:", tx.Key())
	}
	return response, nil
}

func checkSign(uri, token string, data interface{}) (ok bool, msg string) {
	if config.MOD == ModDebug {
		return true, ""
	}
	key := []string{}
	params := map[string]string{}
	switch data.(type) {
	case url.Values:
		for k, v := range data.(url.Values) {
			params[k] = v[0]
			key = append(key, k)
		}
	case map[string]string:
		for k := range data.(map[string]string) {
			key = append(key, k)
		}
		params = data.(map[string]string)
	default:
		return false, "type error"
	}
	sign, ok := params["sign"]
	if !ok {
		return false, "miss sign"
	}
	sort.Strings(key)
	strs := []string{}
	for _, v := range key {
		if v == "sign" {
			continue
		}
		strs = append(strs, fmt.Sprintf("%s=%v", v, params[v]))
	}
	fullstr := uri + strings.Join(strs, "&")
	return sign == utils.GetMD5(fullstr+token), fullstr
}
