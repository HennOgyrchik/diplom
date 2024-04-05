package fileStorage

import (
	"fmt"
	"github.com/jlaffaye/ftp"
	"io"
	"time"
)

type FileStorageConfig struct {
	connStr  string
	user     string
	password string
}

const (
	timeLayout = "02-01-2006_15-04-05"
)

func New(ftpConn, user, password string) FileStorageConfig {
	return FileStorageConfig{
		connStr:  ftpConn,
		user:     user,
		password: password,
	}
}

func (f FileStorageConfig) conn() (*ftp.ServerConn, error) {
	ftpClient, err := ftp.Dial(f.connStr)
	if err != nil {
		return nil, fmt.Errorf("FTP Connect: %w", err)
	}

	if err = ftpClient.Login(f.user, f.password); err != nil {
		return nil, fmt.Errorf("FTP Login: %w", err)
	}

	return ftpClient, nil
}

func (f FileStorageConfig) StoreFile(fileExt string, r io.Reader) (string, error) {
	conn, err := f.conn()
	if err != nil {
		return "", err
	}
	defer conn.Quit()

	fileName := "Receipt" + "_" + time.Now().Format(timeLayout) + fileExt

	return fileName, conn.Stor(fileName, r)
}

func (f FileStorageConfig) ReadFile(fileName string) ([]byte, error) {
	conn, err := f.conn()
	if err != nil {
		return nil, err
	}
	defer conn.Quit()

	r, err := conn.Retr(fileName)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}
