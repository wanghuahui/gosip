package main

import (
	"fmt"
	"net"
	"time"

	"github.com/panjjo/gosip/sip"
	"github.com/panjjo/gosip/utils"
	"go.mongodb.org/mongo-driver/mongo"
)

const userTB = "users"     // 用户表 NVR表
const deviceTB = "devices" // 设备表 摄像头

// NVRDevices NVR  设备信息
type NVRDevices struct {
	DBModel
	// Name 设备名称
	Name string `json:"name" bson:"name"`
	// DeviceID 设备id
	DeviceID string `json:"deviceid" bson:"deviceid"`
	// Region 设备域
	Region string `json:"region" bson:"region"`
	// Host Via 地址
	Host string `json:"host" bson:"host"`
	// Port via 端口
	Port string `json:"port" bson:"port"`
	// TransPort via transport
	TransPort string `json:"transport" bson:"transport"`
	// Proto 协议
	Proto string `json:"proto" bson:"proto"`
	// Rport via rport
	Rport string `json:"report" bson:"report"`
	// RAddr via recevied
	RAddr string `json:"raddr" bson:"raddr"`
	// Manufacturer 制造厂商
	Manufacturer string `xml:"Manufacturer" bson:"manufacturer" json:"manufacturer"`
	// 设备类型DVR，NVR
	DeviceType string `xml:"DeviceType" bson:"devicetype" json:"devicetype"`
	// Firmware 固件版本
	Firmware string `bson:"firmware" json:"firmware"`
	// Model 型号
	Model  string `bson:"model" json:"model"`
	URIStr string `json:"uri" bson:"uri"`
	// ActiveAt 最后心跳检测时间
	ActiveAt int64 `json:"active" bson:"active"`
	// Regist 是否注册
	Regist bool `json:"regist" bson:"regist"`
	// PWD 密码
	PWD string  `json:"pwd" bson:"pwd"`
	Sys sysInfo `json:"sysinfo" bson:"-"`
	//----
	addr   *sip.Address
	source net.Addr
}

// Devices 摄像头信息
type Devices struct {
	DBModel
	// DeviceID 设备编号
	DeviceID string `xml:"DeviceID" bson:"deviceid" json:"deviceid"`
	// Name 设备名称
	Name         string `xml:"Name" bson:"name" json:"name"`
	Manufacturer string `xml:"Manufacturer" bson:"manufacturer" json:"manufacturer"`
	Model        string `xml:"Model" bson:"model" json:"model"`
	Owner        string `xml:"Owner" bson:"owner" json:"owner"`
	CivilCode    string `xml:"CivilCode" bson:"civilcode" json:"civilcode"`
	// Address ip地址
	Address     string `xml:"Address" bson:"address" json:"address"`
	Parental    int    `xml:"Parental" bson:"parental" json:"parental"`
	SafetyWay   int    `xml:"SafetyWay" bson:"safetyway" json:"safetyway"`
	RegisterWay int    `xml:"RegisterWay" bson:"registerway" json:"registerway"`
	Secrecy     int    `xml:"Secrecy" bson:"secrecy" json:"secrecy"`
	// Status 状态  on 在线
	Status string `xml:"Status" bson:"status" json:"status"`
	// PDID 所属用户id
	PDID string `bson:"pdid" json:"pdid"`
	// Active 最后活跃时间
	Active int64  `bson:"active" json:"active"`
	URIStr string `bson:"uri" json:"uri"`
}

// 从请求中解析出设备信息
func parserDevicesFromReqeust(req *sip.Request) (NVRDevices, bool) {
	u := NVRDevices{}
	header, ok := req.From()
	if !ok {
		logger.Warn("not found from header from request", req.String())
		return u, false
	}
	if header.Address == nil {
		logger.Warn("not found from user from request", req.String())
		return u, false
	}
	if header.Address.User() == nil {
		logger.Warn("not found from user from request", req.String())
		return u, false
	}
	u.DeviceID = header.Address.User().String()
	u.Region = header.Address.Host()
	via, ok := req.ViaHop()
	if !ok {
		logger.Info("not found ViaHop from request", req.String())
		return u, false
	}
	u.Host = via.Host
	u.Port = via.Port.String()
	report, ok := via.Params.Get("rport")
	if ok && report != nil {
		u.Rport = report.String()
		// } else {
		// 	u.Rport = u.Port
	}
	raddr, ok := via.Params.Get("received")
	if ok && raddr != nil {
		u.RAddr = raddr.String()
		// } else {
		// 	u.RAddr = u.Host
	}

	u.TransPort = via.Transport
	u.URIStr = header.Address.String()
	u.addr = sip.NewAddressFromFromHeader(header)
	u.source = req.Source()
	return u, true
}

// 获取设备信息（注册设备）
func sipDeviceInfo(to NVRDevices) {
	var fromaddr = _serverDevices.addr
	fromaddr.Params.Add("tag", sip.String{Str: utils.RandString(20)})
	hb := sip.NewHeaderBuilder().SetTo(to.addr).SetFrom(fromaddr).AddVia(&sip.ViaHop{
		Params: sip.NewParams().Add("branch", sip.String{Str: sip.GenerateBranch()}),
	}).SetContentType(&sip.ContentTypeXML).SetMethod(sip.MESSAGE)
	req := sip.NewRequest("", sip.MESSAGE, to.addr.URI, sip.DefaultSipVersion, hb.Build(), sip.GetDeviceInfoXML(to.DeviceID))
	req.SetDestination(to.source) // 相机设备地址，IP:Port
	tx, err := srv.Request(req)
	// fmt.Printf("sip device request: %+v\n", req.String())
	if err != nil {
		logger.Warn("sipDeviceInfo error, ", err)
		return
	}
	_, err = sipResponse(tx)
	if err != nil {
		logger.Warn("sipDeviceInfo response error, ", err)
		return
	}
}

// sipCatalog 获取注册设备包含的列表
func sipCatalog(to NVRDevices) {
	var fromaddr = _serverDevices.addr
	fromaddr.Params.Add("tag", sip.String{Str: utils.RandString(20)})

	hb := sip.NewHeaderBuilder().SetTo(to.addr).SetFrom(fromaddr).AddVia(&sip.ViaHop{
		Params: sip.NewParams().Add("branch", sip.String{Str: sip.GenerateBranch()}),
	}).SetContentType(&sip.ContentTypeXML).SetMethod(sip.MESSAGE)
	req := sip.NewRequest("", sip.MESSAGE, to.addr.URI, sip.DefaultSipVersion, hb.Build(), sip.GetCatalogXML(to.DeviceID))
	req.SetDestination(to.source)
	tx, err := srv.Request(req)
	// fmt.Printf("sip catalog request: %+v\n", req.String())
	if err != nil {
		logger.Warn("sipCatalog  error,", err)
		return
	}
	_, err = sipResponse(tx)
	if err != nil {
		logger.Warn("sipCatalog response error,", err)
		return
	}
}

// MessageDeviceInfoResponse 主设备明细返回结构
type MessageDeviceInfoResponse struct {
	CmdType      string `xml:"CmdType"`
	SN           int    `xml:"SN"`
	DeviceID     string `xml:"DeviceID"`
	DeviceType   string `xml:"DeviceType"`
	Manufacturer string `xml:"Manufacturer"`
	Model        string `xml:"Model"`
	Firmware     string `xml:"Firmware"`
}

func sipMessageDeviceInfo(u NVRDevices, body string) error {
	message := &MessageDeviceInfoResponse{}
	if err := utils.XMLDecode([]byte(body), message); err != nil {
		logger.Error("sipMessageDeviceInfo Unmarshal xml err:", err, "body:", body)
		return err
	}
	update := M{
		"model":        message.Model,
		"devicetype":   message.DeviceType,
		"firmware":     message.Firmware,
		"manufacturer": message.Manufacturer,
	}
	dbClient.Update(userTB, M{"deviceid": u.DeviceID}, M{"$set": update})
	return nil
}

// MessageDeviceListResponse 设备明细列表返回结构
type MessageDeviceListResponse struct {
	CmdType  string       `xml:"CmdType"`
	SN       int          `xml:"SN"`
	DeviceID string       `xml:"DeviceID"`
	SumNum   int          `xml:"SumNum"`
	Item     []DeviceItem `xml:"DeviceList>Item"`
}

// DeviceItem 设备明细结构
type DeviceItem struct {
	// DeviceID 设备编号
	DeviceID string `xml:"DeviceID" bson:"deviceid" json:"deviceid"`
	// Name 设备名称
	Name         string `xml:"Name" bson:"name" json:"name"`
	Manufacturer string `xml:"Manufacturer" bson:"manufacturer" json:"manufacturer"`
	Model        string `xml:"Model" bson:"model" json:"model"`
	Owner        string `xml:"Owner" bson:"owner" json:"owner"`
	CivilCode    string `xml:"CivilCode" bson:"civilcode" json:"civilcode"`
	// Address ip地址
	Address     string `xml:"Address" bson:"address" json:"address"`
	Parental    int    `xml:"Parental" bson:"parental" json:"parental"`
	SafetyWay   int    `xml:"SafetyWay" bson:"safetyway" json:"safetyway"`
	RegisterWay int    `xml:"RegisterWay" bson:"registerway" json:"registerway"`
	Secrecy     int    `xml:"Secrecy" bson:"secrecy" json:"secrecy"`
	// Status 状态  on 在线
	Status string `xml:"Status" bson:"status" json:"status"`
	// PDID 所属用户id
	PDID string `bson:"pdid" json:"pdid"`
	// Active 最后活跃时间
	Active int64  `bson:"active" json:"active"`
	URIStr string `bson:"uri" json:"uri"`

	addr *sip.Address `bson:"-"`
}

func sipMessageCatalog(u NVRDevices, body string) error {
	message := &MessageDeviceListResponse{}
	if err := utils.XMLDecode([]byte(body), message); err != nil {
		logger.Error("Message Unmarshal xml err:", err, "body:", body)
		return err
	}
	if message.SumNum > 0 {
		device := &DeviceItem{}
		var err error
		for _, d := range message.Item {
			if err = dbClient.Get(deviceTB, M{"deviceid": d.DeviceID, "pdid": message.DeviceID}, device); err == nil {
				d.PDID = message.DeviceID
				d.Active = time.Now().Unix()
				d.URIStr = fmt.Sprintf("sip:%s@%s", d.DeviceID, _sysinfo.Region)
				d.Status = transDeviceStatus(d.Status)
				dbClient.Update(deviceTB, M{"deviceid": d.DeviceID, "pdid": d.PDID}, M{"$set": d})
			} else {
				if err != mongo.ErrNoDocuments {
					logger.Errorf("Get Device Info failed, deviceid:%s, pdid:%s, err:%s", d.DeviceID, message.DeviceID, err.Error())
					return err
				}

				d.PDID = message.DeviceID
				d.Active = time.Now().Unix()
				d.URIStr = fmt.Sprintf("sip:%s@%s", d.DeviceID, _sysinfo.Region)
				d.Status = transDeviceStatus(d.Status)
				if err = dbClient.Insert(deviceTB, d); err != nil {
					logger.Error("Insert Device Info failed, err: ", err.Error())
					return err
				}
				if err = dbClient.Update(SysInfoTbl, M{}, M{"$inc": M{"dnum": 1}}); err != nil {
					logger.Error("Update Sysinfo failed, err: ", err.Error())
					return err
				}
				_sysinfo.DNUM++
			}
		}
	}
	return nil
}

var deviceStatusMap = map[string]string{
	"ON":     "ON",
	"ONLINE": "ON",
	"OFFILE": "OFF",
	"OFF":    "OFF",
}

func transDeviceStatus(status string) string {
	if v, ok := deviceStatusMap[status]; ok {
		return v
	}
	return status
}
