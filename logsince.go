/**
logsince

Print log from last line printed, creates hidden .logsince and .logsince.LCK files

Created by Him You Ten on 2014-04-03.

The MIT License (MIT)

Copyright (c) 2014 himyouten

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
**/

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"bufio"
	"io"
	"strings"
)

var logfile string
var debug bool = false
var test bool = false
var start int64 = -1
var length int = -1
var clean bool = false
var lastposfile string = ""
var lockfile string = ""

func FileExists(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}

	return true
}

func PrintDebug(format string, a ...interface{}) {
	if debug {
		fmt.Printf(format+"\n", a...)
	}
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n%[1]s [options] LOGFILE\n\nWhere:\n", filepath.Base(os.Args[0])) 
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nLOGFILE: logfile to process\n") 
		os.Exit(0)
	}
	parseArgs()
}

func parseArgs() {
	help := false

	flag.BoolVar(&help, "help", false, "show this help")
	flag.Int64Var(&start, "start", -1, "use start instead of last position, in bytes")
	flag.IntVar(&length, "length", -1, "number of lines to read, defaults to end of file")
	flag.BoolVar(&clean, "clean", false, "clean up hidden files")
	flag.BoolVar(&debug, "debug", false, "turn debug on")
	flag.BoolVar(&test, "test", false, "do not write to .logsince file, only read from it")

	flag.Parse()

	if help {
		flag.Usage()
	}

	if len(flag.Args()) > 0 {
		logfile = flag.Arg(0)
	}
	if len(logfile) == 0 {
		fmt.Printf("ERROR: missing logfile\n")
		os.Exit(1)
	}

	dirname, filename := filepath.Split(logfile)
	PrintDebug("DEBUG: dirname:%v filename:%v", dirname, filename)

	if len(dirname) > 0 {
		dirname += "/"
	}

	lastposfile = dirname + getLastposfile(filename)
	lockfile = dirname + getLockfile(filename)

}

func getLastposfile(logfile string) string {
	// Return the lastpos file, .logfile_name.logsince
	return "." + logfile + ".logsince"
}

func getLockfile(logfile string) string {
	// Return the lock file, .logfile_name.logsince.LCK
	return getLastposfile(logfile) + ".LCK"
}

func getBakfile(lastpostfile string) string {
	// Return the backup file, .logfile_name.logsince.bak
	return lastpostfile + ".bak"
}

func panicOnError(e error) {
	if e != nil {
		panic(e)
	}
}

func writeLastsize(lastposfile string, filesize int64) {
	// Writes the last size to the logsince file
	if test {
		PrintDebug("DEBUG: Test set, not writing")
		return
	}
	err := ioutil.WriteFile(lastposfile, []byte(strconv.FormatInt(filesize, 10)+"\n"), 0644)
	panicOnError(err)
	PrintDebug("DEBUG: new %v created", lastposfile)
}

func writeLastpos(lastposfile string, lastpos int64) {
	// Writes the last pos to the logsince file
	if test {
		PrintDebug("DEBUG: Test set, not writing")
		return
	}

	PrintDebug("DEBUG: saving lastpos %v to %v", lastpos, lastposfile)
	f, err := os.OpenFile(lastposfile, os.O_WRONLY|os.O_APPEND, 0660)
	panicOnError(err)
	defer f.Close()
	_, err = f.WriteString(strconv.FormatInt(lastpos, 10))
	panicOnError(err)
	f.Sync()

	PrintDebug("DEBUG: %v updated with last pos %v", lastposfile, lastpos)
}

func getLastpos(logfile string, lastposfile string, start int64, by_byte bool) int64 {
	// Read last location from .filename.logsince hidden file.
	// Ignore if start is > -1 and use start instead
	// If size is smaller, start from the beginning

	var default_startpos int64 = 1
	// if we're doing by bytes and not line count, then start at 0
	if by_byte {
		default_startpos = 0
	}

	var lastpos int64 = 1
	filestat, err := os.Stat(logfile)
	panicOnError(err)

	// if start > -1, return start
	if start > -1 {
		PrintDebug("DEBUG: Start set, starting at %v", start)
		writeLastsize(lastposfile, filestat.Size())
		return start
	}
	// read lastpos file
	PrintDebug("DEBUG: Checking last pos from %v", lastposfile)
	// if not found, return 0
	if !FileExists(lastposfile) {
		PrintDebug("DEBUG: Last pos file not found")
		writeLastsize(lastposfile, filestat.Size())
		return default_startpos
	}
	// get last pos and last size from file
	f, err := os.Open(lastposfile)
	panicOnError(err)
	defer f.Close()
	r := bufio.NewReader(f)
	lastsize_str, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		panic(err)
	}
	lastsize_str = strings.TrimSpace(lastsize_str)
	lastpos_str, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		panic(err)
	}
	lastpos_str = strings.TrimSpace(lastpos_str)

	// check if file is smaller, if so start at the beginning
	lastsize, err := strconv.ParseInt(lastsize_str, 10, 64)
	if err != nil {
		lastsize = 0
		PrintDebug("DEBUG: Last size found cannot be converted to integer, '%v'", lastsize_str)
	}

	writeLastsize(lastposfile, filestat.Size())
	if filestat.Size() < lastsize {
		PrintDebug("DEBUG: File is smaller, %v vs %v, starting from beginning", filestat.Size(), lastsize)
		return default_startpos
	}

	lastpos, err = strconv.ParseInt(lastpos_str, 10, 64)
	if by_byte {
		lastpos += 1
	}
	if err != nil {
		lastpos = default_startpos
		PrintDebug("DEBUG: Last pos found cannot be converted to integer, '%v'", lastpos_str)
	}
	PrintDebug("DEBUG: Starting at %v", lastpos)

	return lastpos
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src string, dst string, hardlink bool) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %v (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %v (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if hardlink {
		if err = os.Link(src, dst); err == nil {
			return
		}
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
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

func backupfile(lastposfile string) (err error) {
	// backup the pos file
	if FileExists(lastposfile) {
		PrintDebug("DEBUG: Backing up %v ...", lastposfile)
		err := CopyFile(lastposfile, getBakfile(lastposfile), false)
		return err
	}
	return
}

func cleanBackupfile(lastposfile string) (err error) {
	// backup the pos file
	bakfile := getBakfile(lastposfile)
	if FileExists(bakfile) {
		PrintDebug("DEBUG: Cleaning backup %v", bakfile)
		err := os.Remove(bakfile)
		return err
	}
	return
}

func lock(lockfile string) (bool, error) {
	// Creates the lock file, if already exists, returns None, otherwise stores the pid and returns the file name
	PrintDebug("DEBUG: Locking with %v", lockfile)
	if FileExists(lockfile) {
		PrintDebug("DEBUG: Lock found")
		return false, nil
	}
	// create the lock
	pid := os.Getpid()
	err := ioutil.WriteFile(lockfile, []byte(strconv.Itoa(pid)+"\n"), 0644)
	if err != nil {
		return false, err
	}
	PrintDebug("DEBUG: Wrote %v to lockfile", pid)
	return true, nil
}

func unlock(lockfile string) (bool, error) {
	// Deletes the lockfile, if no file exists, returns false
	PrintDebug("DEBUG: Unlocking %v", lockfile)
	if !FileExists(lockfile) {
		return false, nil
	}
	err := os.Remove(lockfile)
	if err != nil {
		return false, err
	}
	return true, nil
}

func println(logfile string, lastpos int64, length int, lastposfile string) {
	// Print using go, use seek to move the the correct location, save the last position
	// print the logfile, get the last position
	new_lastpos := printLogfile(logfile, lastpos, length)
	// write the last position to file
	writeLastpos(lastposfile, new_lastpos)
}

func printLogfile(logfile string, lastpos int64, length int) int64 {
	// Print using go, use seek to move the the correct location
	// open the logfile
	linecount := 0
	f, err := os.Open(logfile)
	panicOnError(err)
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	// seek to the lastpos
	_, err = f.Seek(lastpos, os.SEEK_SET)

	r := bufio.NewReader(f)

	moreLines := true
	var bytesRead int64 = 0
	for moreLines {
		// read the line
		line, err := r.ReadString('\n')
		// check if we should be continuing
		if err == nil {
			moreLines = true
		} else if err == io.EOF {
			// no more data
			moreLines = false
		} else {
			// whoa! that's not good!
			panicOnError(err)
		}
		fmt.Print(line)
		linecount += 1
		// stop if count reaches length
		if length > -1 && linecount >= length {
			PrintDebug("DEBUG: Length reached, stopping at line count %v", linecount)
			moreLines = false
		}
		bytesRead += int64(len(line))
		PrintDebug("bytesRead:%v", bytesRead)
	}
	// if nothing read - move pointer back
	if bytesRead == 0 {
		bytesRead = -1
	}

	return lastpos + bytesRead
}

func main() {
	// Gets the arguments, locks the file, gets the last position and runs the sed cmd
	// Unlocks when done

	if clean {
		cleanFiles := [3]string{lastposfile, lockfile, getBakfile(lastposfile)}
		PrintDebug("DEBUG: Cleaning up logsince hidden files")
		for _,f := range cleanFiles {
			if FileExists(f) {
				os.Remove(f)
			}
		}
		os.Exit(0)
	}

	// lock the file
	if locked, err := lock(lockfile); !locked || err != nil {
		// TODO print to stderr
		fmt.Printf("ERROR: Cannot create lock for %v", logfile)
		os.Exit(1)
	}

	// unlock - defered
	defer func() {
		if _, err := unlock(lockfile); err != nil {
			panic(err)
		}
	}()

	// backup
	err := backupfile(lastposfile)
	panicOnError(err)

	// clean bak - defered
	defer func() {
		if err := cleanBackupfile(lastposfile); err != nil {
			panic(err)
		}
	}()

	// read last location from .filename.logsince hidden file - ignore if start is > -1 and use start instead
	lastpos := getLastpos(logfile, lastposfile, start, true)
	println(logfile, lastpos, length, lastposfile)

}
