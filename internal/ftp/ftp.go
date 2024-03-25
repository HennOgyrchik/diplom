package ftp

import (
	"github.com/jlaffaye/ftp"
	"io"
)

type FTP string

func serverConnection(address FTP) (client *ftp.ServerConn, err error) {
	client, err = ftp.Dial(string(address))
	if err != nil {
		return
	}

	if err = client.Login("user", "123"); err != nil {
		return
	}

	return
}

func (ftp FTP) StoreFile(fileName string, r io.Reader) error {
	client, err := serverConnection(ftp)
	if err != nil {
		return err
	}
	defer client.Quit()

	return client.Stor(fileName, r)
}

func (ftp FTP) ReadFile(fileName string) ([]byte, error) {
	client, err := serverConnection(ftp)
	if err != nil {
		return nil, err
	}
	defer client.Quit()

	r, err := client.Retr(fileName)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	return io.ReadAll(r)
}
