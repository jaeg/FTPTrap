package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/sftp"
)

//JunkHandler handler struct for request server
type JunkHandler struct {
	commandDelay int64
	user         string
	ip           string
}

//Fileread handler for read requests for the sftp server.
func (fs *JunkHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	logActivity(fs.ip, fs.user, "File Read Operation: "+r.Method+" , Path: "+r.Filepath)

	file, ok := junkConfig.JunkFiles[r.Filepath]
	fmt.Println(r.Filepath)
	if ok {
		return strings.NewReader(file.Content), nil
	}

	return strings.NewReader("hello world"), nil
}

//Filewrite handler for write requests for the sftp server.
func (fs *JunkHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	logActivity(fs.ip, fs.user, "File Write Operation: "+r.Method+" , Path: "+r.Filepath)

	return nil, nil
}

//Filecmd handler for commands for the sftp server.
func (fs *JunkHandler) Filecmd(r *sftp.Request) error {
	logActivity(fs.ip, fs.user, "File Command Operation: "+r.Method+" , Path: "+r.Filepath)

	time.Sleep(time.Second * time.Duration(fs.commandDelay))

	return nil
}

type listerat []os.FileInfo

// Modeled after strings.Reader's ReadAt() implementation
func (f listerat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(f)) {
		return 0, io.EOF
	}
	n = copy(ls, f[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}

//Filelist handler for list commands for the sftp server.
func (fs *JunkHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	logActivity(fs.ip, fs.user, "File List Operation: "+r.Method+" , Path: "+r.Filepath)
	time.Sleep(time.Second * time.Duration(fs.commandDelay))
	switch r.Method {
	case "List":
		files := make([]os.FileInfo, 0)
		for _, v := range junkConfig.JunkFiles {
			info := JunkFile{FileName: v.FileName}
			files = append(files, info)
		}

		return listerat(files), nil
	case "Stat":
		fmt.Println("Statting")
		info := JunkFile{FileName: r.Filepath}
		return listerat([]os.FileInfo{info}), nil
	case "Readlink":
		return nil, nil
	}
	return nil, nil
}

// GetJunkHandler returns a Hanlders object with the test handlers.
func GetJunkHandler(user string, ip string, commandDelay int64) (sftp.Handlers, error) {
	fileWriter := &JunkHandler{commandDelay: commandDelay, user: user, ip: ip}

	return sftp.Handlers{FileGet: fileWriter, FilePut: fileWriter, FileCmd: fileWriter, FileList: fileWriter}, nil
}
