package config

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	WebDAV   WebDAVConfig   `mapstructure:"webdav"`
	Scan     ScanConfig     `mapstructure:"scan"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type WebDAVConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	RootPath string `mapstructure:"root_path"`
}

type ScanConfig struct {
	Extensions []string `mapstructure:"extensions"`
	BatchSize  int      `mapstructure:"batch_size"`
	Concurrent int      `mapstructure:"concurrent"`
}

var (
	cfg  *Config
	once sync.Once
)

func GetConfig() *Config {
	once.Do(func() {
		cfg = &Config{}
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./")

		if err := viper.ReadInConfig(); err != nil {
			fmt.Printf("Warning: config file not found, using defaults: %v\n", err)
			setDefaults()
		}

		if err := viper.Unmarshal(cfg); err != nil {
			panic(fmt.Sprintf("Failed to unmarshal config: %v", err))
		}
	})
	return cfg
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.path", "./data/music.db")
	viper.SetDefault("webdav.url", "http://192.168.1.3:9000")
	viper.SetDefault("webdav.username", "music")
	viper.SetDefault("webdav.password", "musci")
	viper.SetDefault("webdav.root_path", "/dav")
	viper.SetDefault("scan.extensions", []string{".mp3"})
	viper.SetDefault("scan.batch_size", 50)
	viper.SetDefault("scan.concurrent", 5)
}
