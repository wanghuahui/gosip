package main

import (
	"strings"

	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"
)

// Config Config
type Config struct {
	MOD      string        `json:"mod" yaml:"mod" mapstructure:"mod"`
	DB       DBConfig      `json:"database" yaml:"database" mapstructure:"database"`
	LogLevel string        `json:"logger" yaml:"logger" mapstructure:"logger"`
	UDP      string        `json:"udp" yaml:"udp" mapstructure:"udp"`
	API      string        `json:"api" yaml:"api" mapstructure:"api"`
	Secret   string        `json:"secret" yaml:"secret" mapstructure:"secret"`
	Media    []MediaServer `json:"media" yaml:"media" mapstructure:"media"`
	Stream   Stream        `json:"stream" yaml:"stream" mapstructure:"stream"`
	MP4Path  string        `json:"mp4path" yaml:"mp4path" mapstructure:"mp4path"`
	GB28181  sysInfo       `json:"gb28181" yaml:"gb28181" mapstructure:"gb28181"`
}

// Stream Stream
type Stream struct {
	HLS       bool  `json:"hls" yaml:"hls" mapstructure:"hls"`
	RTMP      bool  `json:"rtmp" yaml:"rtmp" mapstructure:"rtmp"`
	RecordMax int64 `json:"recordmax" yaml:"recordmax" mapstructure:"recordmax"`
}

// MediaServer MediaServer
type MediaServer struct {
	RESTFUL string `json:"restful" yaml:"restful" mapstructure:"restful"`
	HTTP    string `json:"http" yaml:"http" mapstructure:"http"`
	WS      string `json:"ws" yaml:"ws" mapstructure:"ws"`
	RTMP    string `json:"rtmp" yaml:"rtmp" mapstructure:"rtmp"`
	RTSP    string `json:"rtsp" yaml:"rtsp" mapstructure:"rtsp"`
	RTP     string `json:"rtp" yaml:"rtp" mapstructure:"rtp"`
	Secret  string `json:"secret" yaml:"secret" mapstructure:"secret"`
}

var config *Config

// parseLevel 解析日志打印等级
func parseLevel(level string) (l log.Lvl) {
	switch level {
	case "debug":
		l = log.DEBUG
	case "info":
		l = log.INFO
	case "warn":
		l = log.WARN
	case "error":
		l = log.ERROR
	case "off":
		l = log.OFF
	default:
		l = log.INFO
	}
	return
}

func loadConfig() {
	viper.SetConfigType("yml")
	viper.SetConfigName("config")
	viper.AddConfigPath("./")
	viper.SetDefault("logger", "debug")
	viper.SetDefault("udp", "0.0.0.0:5060")
	viper.SetDefault("api", "0.0.0.0:8090")
	viper.SetDefault("mod", "release")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatal("init config error:", err)
	}
	logger.Info("read config ok")

	config = &Config{}
	err = viper.Unmarshal(&config)
	if err != nil {
		logger.Fatal("init config unmarshal error:", err)
	}
	logger.Infof("config :%+v", config)

	// level, _ := logger.ParseLevel(config.LogLevel)
	// logger.SetLevel(level)
	logger.SetLevel(parseLevel(config.LogLevel))
	InitDB(config.DB)
	config.MOD = strings.ToUpper(config.MOD)
}
