package util

import (
	"fmt"
	"strconv"
	"time"
)

func GetFullTimeString(format string) string {
	timeObj := time.Now()

	yy := timeObj.Year()
	yyStr := strconv.Itoa(yy)

	mm := int(timeObj.Month())
	mmStr := strconv.Itoa(mm)
	if mm < 10 {
		mmStr = "0" + mmStr
	}

	dd := timeObj.Day()
	ddStr := strconv.Itoa(dd)
	if dd < 10 {
		ddStr = "0" + ddStr
	}

	h := timeObj.Hour()
	hStr := strconv.Itoa(h)
	if h < 10 {
		hStr = "0" + hStr
	}

	m := timeObj.Minute()
	mStr := strconv.Itoa(m)
	if m < 10 {
		mStr = "0" + mStr
	}

	s := timeObj.Second()
	sStr := strconv.Itoa(s)
	if s < 10 {
		sStr = "0" + sStr
	}

	return fmt.Sprintf(format, yyStr, mmStr, ddStr, hStr, mStr, sStr)
}

func GetDateString(format string) string {
	timeObj := time.Now()

	yy := timeObj.Year()
	yyStr := strconv.Itoa(yy)

	mm := int(timeObj.Month())
	mmStr := strconv.Itoa(mm)
	if mm < 10 {
		mmStr = "0" + mmStr
	}

	dd := timeObj.Day()
	ddStr := strconv.Itoa(dd)
	if dd < 10 {
		ddStr = "0" + ddStr
	}

	return fmt.Sprintf(format, yyStr, mmStr, ddStr)
}

func GetTimeString(format string) string {
	timeObj := time.Now()

	h := timeObj.Hour()
	hStr := strconv.Itoa(h)
	if h < 10 {
		hStr = "0" + hStr
	}

	m := timeObj.Minute()
	mStr := strconv.Itoa(m)
	if m < 10 {
		mStr = "0" + mStr
	}

	s := timeObj.Second()
	sStr := strconv.Itoa(s)
	if s < 10 {
		sStr = "0" + sStr
	}

	return fmt.Sprintf(format, hStr, mStr, sStr)
}
