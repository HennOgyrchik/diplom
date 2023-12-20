package ftp

import (
	"github.com/jlaffaye/ftp"
	"io"
	"io/ioutil"
)

func FTPServerConnection() (client *ftp.ServerConn, err error) {
	client, err = ftp.Dial("192.168.0.103:21")
	if err != nil {
		return
	}

	if err = client.Login("user", "123"); err != nil {
		return
	}

	return
}

func StoreFile(fileName string, r io.Reader) (ok bool, err error) {
	ok = false

	client, err := FTPServerConnection()
	if err != nil {
		return
	}
	defer client.Quit()

	//data := bytes.NewBufferString("Hello World")
	err = client.Stor(fileName, r)
	if err != nil {
		panic(err)
	}
	ok = true
	return
}

func ReadFile() {
	client, err := FTPServerConnection()
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
