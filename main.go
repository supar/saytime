package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Time struct {
	time.Time
}

type Delta struct {
	sync.Mutex
	time.Duration
	Year, Month, Day int
}

// ResponseIface represents interface to work with
// response data
type ResponseIface interface {
	Get() ([]byte, error)
	Ok() bool
	Status() int
}

type Response struct {
	Time  *Time  `json:"time,omitempty"`
	Error *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
}

type Controller func(*http.Request) ResponseIface

const (
	DateFloatFormat = "060102.150405"
)

var (
	ErrDateFormat = errors.New("unexpected date format YYMMDD")

	GlobalDelta = Delta{}
)

func main() {

}

func timeNow(r *http.Request) ResponseIface {
	tm := NewTime().Delta(GlobalDelta)

	return Response{
		Time: &tm,
	}
}

func timeAdd(r *http.Request) ResponseIface {
	var (
		dateTime Time
		delta    Delta
		err      error
	)

	if dateTime, err = parseDateTime([]byte(r.URL.Query().Get("time"))); err != nil {
		return Response{
			Error: &Error{
				Code:    500,
				Message: err.Error(),
			},
		}
	}

	if delta, err = parseDelta([]byte(r.URL.Query().Get("delta"))); err != nil {
		return Response{
			Error: &Error{
				Code:    500,
				Message: err.Error(),
			},
		}
	}

	tm := dateTime.Delta(delta)

	return Response{
		Time: &tm,
	}
}

func timeSet(r *http.Request) ResponseIface {
	tm, err := parseDateTime([]byte(r.PostFormValue("time")))

	if err != nil {
		return Response{
			Error: &Error{
				Code:    500,
				Message: err.Error(),
			},
		}
	}

	GlobalDelta.Set(tm.Sub(time.Now()))

	return Response{}
}

func timeRest(r *http.Request) ResponseIface {
	GlobalDelta.Set(0)
	return Response{}
}

// Create http handler
func NewHandler(fn Controller) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			data     []byte
			response ResponseIface
		)

		if response = fn(r); response == nil {
			return
		}

		data, _ = response.Get()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if !response.Ok() {
			w.WriteHeader(response.Status())
		}

		w.Write(data)
	})
}

func NewTime() Time {
	return Time{
		Time: time.Now(),
	}
}

func (tm Time) Delta(dl Delta) Time {
	return Time{
		tm.Add(dl.Duration).AddDate(dl.Year, dl.Month, dl.Day),
	}
}

// Get returns encoded data ready to send to client
func (s Response) Get() (data []byte, err error) {
	return json.Marshal(s)
}

// Ok returns response state, if Success field is false
func (s Response) Ok() bool {
	return (s.Error == nil)
}

// Status returns response code, default 200
func (s Response) Status() int {
	if !s.Ok() {
		return s.Error.Code
	}

	return 200
}

func (t Time) MarshalJSON() ([]byte, error) {
	var tm = make([]byte, 0, len(DateFloatFormat)+2)

	tm = append(tm, '"')
	tm = t.AppendFormat(tm, DateFloatFormat)
	tm = append(tm, '"')

	return tm, nil
}

func (tm *Time) UnmarshalJSON(data []byte) (err error) {
	(*tm), err = parseDateTime(data)
	return
}

func (dl *Delta) UnmarshalJSON(data []byte) (err error) {
	(*dl), err = parseDelta(data)
	return
}

func (dl *Delta) Set(delta time.Duration) {
	dl.Lock()
	dl.Duration = delta
	dl.Year = 0
	dl.Month = 0
	dl.Day = 0
	dl.Unlock()
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
		fdata    float64
		mantissa int64
	)

	if fdata, err = strconv.ParseFloat(string(data), 64); err != nil {
		return
	}

	if fdata == 0 {
		return
	}

	// YYMMDD
	err = each(int64(fdata), func(idx int, n int64) error {
		switch idx {
		case 1:
			dur.Year = int(n)
		case 2:
			dur.Month = int(n)
		case 3:
			dur.Day = int(n)
		}
		return nil
	})

	if err != nil {
		return
	}

	// hhmmss
	mantissa = int64(fdata*1000000) - int64(fdata)*1000000

	if mantissa == 0 {
		return
	}

	err = each(mantissa, func(idx int, n int64) error {
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

	return
}

func each(num int64, fn func(int, int64) error) error {
	for i, l, s := 10000, 0, 1; i > l; i, s = i/100, s+1 {
		y := num / int64(i)

		if err := fn(s, y); err != nil {
			return err
		}

		num = num - y*int64(i)
	}

	return nil
}
