package ftp

import (
	"github.com/jlaffaye/ftp"
	"io"
	"time"
)

type FTP struct {
	Address  string
	User     string
	Password string
}

const (
	timeLayout = "02-01-2006_15-04-05"
)

func serverConnection(ftpConf FTP) (client *ftp.ServerConn, err error) {
	client, err = ftp.Dial(ftpConf.Address)
	if err != nil {
		return
	}

	if err = client.Login(ftpConf.User, ftpConf.Password); err != nil {
		return
	}

	return
}

func (ftp FTP) StoreFile(fileExt string, r io.Reader) (string, error) {
	fileName := "Receipt" + "_" + time.Now().Format(timeLayout) + fileExt

	client, err := serverConnection(ftp)
	if err != nil {
		return "", err
	}
	defer client.Quit()

	return fileName, client.Stor(fileName, r)
}

func (ftp FTP) ReadFile(fileName string) ([]byte, error) {
	client, err := serverConnection(ftp)
	if err != nil {
		return nil, err
	}
	defer client.Quit()

	r, err := client.Retr(fileName)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}
