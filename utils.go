package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/jcelliott/lumber"
)

var logger *lumber.ConsoleLogger

func init() {
	logger = lumber.NewConsoleLogger(lumber.INFO)
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	logger.Debug("%s took %s.", name, elapsed)
}

func isNumber(s string) (int, bool) {
	num, err := strconv.ParseInt(s, 10, 8)
	if err == nil {
		return int(num), true
	}
	return -1, false
}

// getPassword gets masked password
// from http://stackoverflow.com/questions/2137357/getpasswd-functionality-in-go
func getPassword() string {
	if len(RuntimeArgs.Passphrase) > 0 {
		return string(RuntimeArgs.Passphrase)
	}
	fmt.Print("Enter Password: ")
	bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
	password := string(bytePassword)
	return strings.TrimSpace(password)
}

// exists returns whether the given file or directory exists or not
// from http://stackoverflow.com/questions/10510691/how-to-check-whether-a-file-or-directory-denoted-by-a-path-exists-in-golang
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// Comma produces a string form of the given number in base 10 with
// commas after every three orders of magnitude.
//
// e.g. Comma(834142) -> 834,142
// from https://github.com/dustin/go-humanize/blob/master/comma.go
func Comma(v int64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = 0 - v
	}

	parts := []string{"", "", "", "", "", "", ""}
	j := len(parts) - 1

	for v > 999 {
		parts[j] = strconv.FormatInt(v%1000, 10)
		switch len(parts[j]) {
		case 2:
			parts[j] = "0" + parts[j]
		case 1:
			parts[j] = "00" + parts[j]
		}
		v = v / 1000
		j--
	}
	parts[j] = strconv.Itoa(int(v))
	return sign + strings.Join(parts[j:], ",")
}

// HasInternetAccess determines whether or not the internet is accessible
func HasInternetAccess() bool {
	logger.Debug("Checking internet connection...")
	timeout := time.Duration(2 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	_, connectionError := client.Get("http://www.google.com/")
	internetAccess := true
	if connectionError != nil {
		internetAccess = false
		logger.Debug("...Unavailable")
	} else {
		logger.Debug("...OK")
	}
	return internetAccess
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
// from http://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
// from http://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// shred writes random data to the file before erasing it
func shred(fileName string) error {
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	fileData, err := f.Stat()
	if err != nil {
		return err
	}
	b := make([]byte, fileData.Size())
	_, err = rand.Read(b)
	if err != nil {
		return err
	}
	_, err = f.WriteAt(b, 0)
	if err != nil {
		return err
	}
	f.Close()
	err = os.Remove(fileName)
	if err != nil {
		return err
	}
	return nil
}
