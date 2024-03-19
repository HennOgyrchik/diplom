package config

import (
	"encoding/json"
	"io"
	"os"
	"project1/internal/db"
	"project1/internal/ftp"
)

type Config struct {
	DB    db.ConnString
	FTP   ftp.FTP
	Token string
}

func NewConfig() (Config, error) {
	var conf Config
	f, err := os.Open("./internal/config/conf.json")
	if err != nil {
		return conf, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return conf, err
	}

	var tmp struct {
		Token     string `json:"Token"`
		FTP       string `json:"FTP_Address"`
		PAddress  string `json:"PSQL_Address"`
		PDBName   string `json:"PSQL_DBName"`
		PUser     string `json:"PSQL_User"`
		PPassword string `json:"PSQL_Password"`
		PSSLMode  string `json:"PSQL_SSLMode"`
	}
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return conf, err
	}

	conf.Token = tmp.Token
	conf.FTP = ftp.FTP(tmp.FTP)
	conf.DB, err = db.NewDBConnString(tmp.PAddress, tmp.PDBName, tmp.PUser, tmp.PPassword, tmp.PSSLMode)
	return conf, err
}
