package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env             string        `yaml:"env" env-default:"local"`
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl" env-required:"true"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" env-required:"true"`
	GRPC            `yaml:"grpc"`
	Postgres        Postgres `yaml:"postgres"`
}

type GRPC struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type Postgres struct {
	Host           string        `yaml:"host" env-required:"true"`
	Port           int           `yaml:"port" env-default:"5432"`
	User           string        `yaml:"user" env-required:"true"`
	Password       string        `yaml:"password" env-required:"true"`
	DBName         string        `yaml:"dbname" env-required:"true"`
	SSLMode        string        `yaml:"sslmode" env-default:"disable"`
	MaxConns       int32         `yaml:"max_conns" env-default:"10"`
	MinConns       int32         `yaml:"min_conns" env-default:"2"`
	ConnectTimeout time.Duration `yaml:"connect_timeout" env-default:"5s"`
}

func MustLoad() *Config {
	path := fetchConfigPath()
	if path == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exist: " + path)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}

func fetchConfigPath() string {
	var res string

	// --config
	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
