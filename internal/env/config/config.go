package config

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type Config struct {
	Postgres PostgresConfig `env:",prefix=PSQL_" json:",omitempty"`
	FTP      FTPConfig      `env:",prefix=FTP_" json:",omitempty"`
	Token    string         `env:"TOKEN" json:",omitempty"`
}

type PostgresConfig struct {
	Host        string        `env:"HOST,default=localhost" json:",omitempty"`
	Port        int           `env:"PORT,default=5432" json:",omitempty"`
	Name        string        `env:"NAME,default=postgres" json:",omitempty"`
	User        string        `env:"USER,default=postgres" json:",omitempty"`
	Password    string        `env:"PASSWORD,default=postgres" json:",omitempty"`
	SSLMode     string        `env:"SSLMODE,default=disable" json:",omitempty"`
	ConnTimeout int           `env:"CONN_TIMEOUT,default=5" json:",omitempty"`
	DBTimeout   time.Duration `env:"TIMEOUT,default=5s"`
}

func (p PostgresConfig) ConnectionURL() string {
	host := p.Host
	if v := p.Port; v != 0 {
		host = host + ":" + strconv.Itoa(p.Port)
	}

	u := &url.URL{
		Scheme: "postgres",
		Host:   host,
		Path:   p.Name,
	}

	if p.User != "" || p.Password != "" {
		u.User = url.UserPassword(p.User, p.Password)
	}

	q := u.Query()
	if v := p.ConnTimeout; v > 0 {
		q.Add("connect_timeout", strconv.Itoa(v))
	}
	if v := p.SSLMode; v != "" {
		q.Add("sslmode", v)
	}

	u.RawQuery = q.Encode()

	return u.String()
}

type FTPConfig struct {
	Host     string `env:"HOST,default=localhost" json:",omitempty"`
	Port     int    `env:"PORT,default=21" json:",omitempty"`
	User     string `env:"USER" json:",omitempty"`
	Password string `env:"PASSWORD" json:",omitempty"`
}

func (f FTPConfig) ConnectionString() string {
	return fmt.Sprintf("%s:%d", f.Host, f.Port)
}
