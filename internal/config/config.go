package config

import (
	"net"
	"time"
)

type Config struct {
	LogEnviroment           string
	Cert                    Cert
	ProxyServer             ProxyServer
	ApiServer               ApiServer
	GracefulShotdownTimeout time.Duration
}
type ProxyServer struct {
	Address           string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

type ApiServer struct {
	Address                 string
	ReadTimeout             time.Duration
	WriteTimeout            time.Duration
	IdleTimeout             time.Duration
	ReadHeaderTimeout       time.Duration
	GracefulShotdownTimeout time.Duration
}

type Cert struct {
	CACertFile   string
	CAKeyFile    string
	Organization string
}

func MustLoad() *Config {
	var cfg Config

	cfg = Config{
		LogEnviroment: "local",
		ProxyServer: ProxyServer{
			Address:           net.JoinHostPort("0.0.0.0", "62801"),
			ReadTimeout:       4 * time.Second,
			WriteTimeout:      4 * time.Second,
			IdleTimeout:       30 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
		},
		ApiServer: ApiServer{
			Address:           net.JoinHostPort("0.0.0.0", "62802"),
			ReadTimeout:       4 * time.Second,
			WriteTimeout:      4 * time.Second,
			IdleTimeout:       30 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
		},

		Cert: Cert{
			CACertFile:   "./certs/TrueProxyCA.crt",
			CAKeyFile:    "./certs/TrueProxyCA.key",
			Organization: "TrueProxy",
		},
		GracefulShotdownTimeout: 10 * time.Second,
	}

	return &cfg
}
