package main

import (
	"bytes"
	"errors"
	"strconv"
	"time"
)

type Time struct {
	time.Time
}

const (
	DateFloatFormat = "060102.150405"
)

var (
	ErrDateFormat = errors.New("unexpected date foemat YYMMDD")
)

func main() {

}

func (tm *Time) UnmarshalJSON(data []byte) (err error) {
	(*tm), err = ParseFloated(data)
	return
}

func ParseFloated(f []byte) (t Time, err error) {
	var (
		miliPart []byte

		timePart = bytes.Repeat([]byte("0"), 6)
		parts    = bytes.Split(f, []byte("."))
	)

	switch l := len(parts); {
	case l == 0 || l > 2:
		return t, ErrDateFormat

	default:
		if len(parts[0]) != 6 {
			return t, ErrDateFormat
		}

		if len(parts) > 1 {
			if len(parts[1]) > 6 {
				miliPart = append(miliPart, parts[1][6:]...)

				parts[1] = parts[1][:6]
			}

			copy(timePart, parts[1])
			parts[1] = timePart
		} else {
			parts = append(parts, timePart)
		}
	}

	t.Time, err = time.Parse(DateFloatFormat, string(bytes.Join(parts, []byte("."))))

	if len(miliPart) > 0 && err == nil {
		var msec int64

		if msec, err = strconv.ParseInt(string(miliPart), 10, 64); err == nil {
			t.Time = t.Time.Add(time.Millisecond * time.Duration(msec))
		}
	}

	return
}
