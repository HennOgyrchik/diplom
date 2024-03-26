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
		Token       string `json:"Token"`
		FTPAddress  string `json:"FTP_Address"`
		FTPUser     string `json:"FTP_User"`
		FTPPassword string `json:"FTP_Password"`
		PAddress    string `json:"PSQL_Address"`
		PDBName     string `json:"PSQL_DBName"`
		PUser       string `json:"PSQL_User"`
		PPassword   string `json:"PSQL_Password"`
		PSSLMode    string `json:"PSQL_SSLMode"`
	}
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return conf, err
	}

	conf.Token = tmp.Token
	conf.FTP = ftp.FTP{
		Address:  tmp.FTPAddress,
		User:     tmp.FTPUser,
		Password: tmp.FTPPassword,
	}
	conf.DB, err = db.NewDBConnString(tmp.PAddress, tmp.PDBName, tmp.PUser, tmp.PPassword, tmp.PSSLMode)
	return conf, err
}
