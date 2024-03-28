package fileStorage

import (
	"github.com/jlaffaye/ftp"
	"io"
	"time"
)

type FileStorage struct {
	Conn *ftp.ServerConn
}

const (
	timeLayout = "02-01-2006_15-04-05"
)

func New(ftpConn *ftp.ServerConn) *FileStorage {
	return &FileStorage{Conn: ftpConn}
}

func (ftp *FileStorage) StoreFile(fileExt string, r io.Reader) (string, error) {
	fileName := "Receipt" + "_" + time.Now().Format(timeLayout) + fileExt

	return fileName, ftp.Conn.Stor(fileName, r)
}

func (ftp *FileStorage) ReadFile(fileName string) ([]byte, error) {

	r, err := ftp.Conn.Retr(fileName)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

func (ftp *FileStorage) Close() error {
	return ftp.Conn.Quit()
}
