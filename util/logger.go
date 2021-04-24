package util

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

type LogLv int

const (
	LOG_LV_DEBUG LogLv = 0
	LOG_LV_INFO  LogLv = 1
	LOG_LV_WARN  LogLv = 2
	LOG_LV_ERROR LogLv = 3
)

type logger struct {
	level        LogLv
	dumpOpen     bool
	dumpFile     string
	dumpFileSize int
	logs         chan string
	stopDumpEvt  chan bool
	stopSuccEvt  chan bool
}

var Logger *logger = &logger{
	level:        LOG_LV_DEBUG,
	dumpOpen:     false,
	dumpFile:     "",
	dumpFileSize: 0,
	logs:         make(chan string, 1024),
	stopDumpEvt:  make(chan bool, 1),
	stopSuccEvt:  make(chan bool),
}

func (l *logger) SetLevel(lv LogLv) {
	l.level = lv
}

func (l *logger) D(tag string, a ...interface{}) {
	if l.level > LOG_LV_DEBUG {
		return
	}

	l.doLog(tag, "[DEBUG]", a...)
}

func (l *logger) I(tag string, a ...interface{}) {
	if l.level > LOG_LV_INFO {
		return
	}

	l.doLog(tag, "[INFO ]", a...)
}

func (l *logger) W(tag string, a ...interface{}) {
	if l.level > LOG_LV_WARN {
		return
	}

	l.doLog(tag, "[WARN ]", a...)
}

func (l *logger) E(tag string, a ...interface{}) {
	if l.level > LOG_LV_ERROR {
		return
	}

	l.doLog(tag, "[ERROR]", a...)
}

func (l *logger) doLog(tag string, lv string, a ...interface{}) {
	timeStr := GetFullTimeString("[%s/%s/%s %s:%s:%s]")
	msg := fmt.Sprint(a...)

	if !l.dumpOpen {
		fmt.Println(timeStr, "[", tag, "]", lv, msg)
	} else {
		log := fmt.Sprintln(timeStr, "[", tag, "]", lv, msg)
		l.logs <- log
	}
}

func (l *logger) StartDump(file string, dumpFileSize int) {
	l.dumpFile = file
	l.dumpFileSize = dumpFileSize
	l.dumpOpen = true
	go l.dump()
}

func (l *logger) StopDump() {
	l.dumpOpen = false
	l.stopDumpEvt <- true
	<-l.stopSuccEvt
	l.dumpFile = ""
}

func (l *logger) dump() {
	for {
		bEnd, err := l.dumpToFile()
		l.renameDumpFile()

		if bEnd {
			if err == nil {
				l.stopSuccEvt <- true
			}
			break
		}
	}
}

func (l *logger) dumpToFile() (bool, error) {
	f, err := os.OpenFile(l.dumpFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return true, err
	}

	defer f.Close()

	bEnd := false
	w := bufio.NewWriter(f)

	for {
		select {
		case log := <-l.logs:
			w.WriteString(log)
			w.Flush()

			fs, err := os.Stat(l.dumpFile)
			if err == nil && fs.Size() >= int64(l.dumpFileSize) {
				goto Exit0
			}

		case <-l.stopDumpEvt:
			bEnd = true
			l.dumpAll(w)
			goto Exit0
		}
	}

Exit0:
	return bEnd, nil
}

func (l *logger) dumpAll(w *bufio.Writer) {
	defer w.Flush()

	for {
		select {
		case log := <-l.logs:
			w.WriteString(log)

		default:
			goto Exit0
		}
	}

Exit0:
	return
}

func (l *logger) renameDumpFile() {
	dir := path.Dir(l.dumpFile)
	name := path.Base(l.dumpFile)
	ext := path.Ext(name)
	nameOnly := strings.TrimSuffix(name, ext)
	timeStr := GetFullTimeString("_%s%s%s_%s%s%s")
	newName := path.Join(dir, nameOnly+timeStr+ext)
	os.Rename(l.dumpFile, newName)
}
