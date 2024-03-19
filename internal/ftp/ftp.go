package ftp

import (
	"github.com/jlaffaye/ftp"
	"io"
	"io/ioutil"
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

	err = client.Stor(fileName, r)
	if err != nil {
		return err
	}

	return nil
}

func (ftp FTP) ReadFile() {
	client, err := serverConnection(ftp)
	if err != nil {
		return
	}
	defer client.Quit()

	r, err := client.Retr("test-file.txt")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	println(string(buf))
}
