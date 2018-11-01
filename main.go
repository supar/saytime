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

type Delta struct {
	time.Duration
	Year, Month, Day int
}

const (
	DateFloatFormat = "060102.150405"
)

var (
	ErrDateFormat = errors.New("unexpected date format YYMMDD")
)

func main() {

}

func (tm *Time) UnmarshalJSON(data []byte) (err error) {
	(*tm), err = parseDateTime(data)
	return
}

func (dl *Delta) UnmarshalJSON(data []byte) (err error) {
	(*dl), err = parseDelta(data)
	return
}

func parseDateTime(f []byte) (t Time, err error) {
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

func parseDelta(data []byte) (dur Delta, err error) {
	var (
		parts = bytes.Split(data, []byte("."))
	)

	if l := len(parts); l == 0 || l > 2 {
		return dur, ErrDateFormat
	}

	// YYMMDD
	err = each(parts[0], func(idx, n int) error {
		switch idx {
		case 1:
			dur.Year = n
		case 2:
			dur.Month = n
		case 3:
			dur.Day = n
		}
		return nil
	})

	if err != nil {
		return
	}

	// hhmmss
	if len(parts) > 1 {
		switch l := len(parts[1]); {
		// tie short
		case l < 6:
			tmCopy := bytes.Repeat([]byte("0"), 6)
			copy(tmCopy, parts[1][0:])
			parts[1] = tmCopy

		// cut long
		case l > 6:
			var msec int
			if msec, err = strconv.Atoi(string(parts[1][6:])); err != nil {
				return
			}

			dur.Duration = dur.Duration + time.Millisecond*time.Duration(msec)
			parts[1] = parts[1][:6]
		}

		err = each(parts[1], func(idx, n int) error {
			var x time.Duration

			switch idx {
			case 1:
				x = time.Hour * time.Duration(n)
			case 2:
				x = time.Minute * time.Duration(n)
			case 3:
				x = time.Second * time.Duration(n)
			}

			dur.Duration = dur.Duration + x

			return nil
		})
	}

	return
}

func each(dt []byte, fn func(int, int) error) error {
	num, err := strconv.Atoi(string(dt))
	if err != nil {
		return err
	}

	for i, l, s := 10000, 0, 1; i > l; i, s = int(i/100), s+1 {
		y := int(num / i)

		if err := fn(s, y); err != nil {
			return err
		}

		num = num - y*i
	}

	return nil
}
