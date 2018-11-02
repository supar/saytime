package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_DateInputOverflowDetect(t *testing.T) {
	data := []string{
		"",
		".",
		"..",
		"1",
		"12",
		"122",
		"1220",
		"11220",
		"0181220",
	}

	for i, l := 0, len(data); i < l; i++ {
		_, err := parseDateTime([]byte(data[i]))

		if err != ErrDateFormat {
			t.Error("expected overflow error")
		}
	}
}

func Test_DateTimeInput(t *testing.T) {
	data := []struct {
		in, out string
	}{
		{"180213", "180213.000000"},
		{"180213.12", "180213.120000"},
		{"180213.081833", "180213.081833"},
		{"180313.2218332", "180313.221833"},
		{"180313.221800042", "180313.221800"},
	}

	for i, l := 0, len(data); i < l; i++ {
		tm, err := parseDateTime([]byte(data[i].in))

		if err != nil {
			t.Error(err)
		}

		if f := tm.Format(DateFloatFormat); data[i].out != f {
			t.Error("not equal: " + data[i].out + " <=> " + f)
		}

		if len(data[i].in) > 13 {
			nsec := tm.Nanosecond()
			msec, _ := strconv.Atoi(data[i].in[13:])

			if nsec != (int(time.Millisecond) * msec) {
				t.Error("invalid miliseconds value")
			}
		}
	}
}

func Test_UnmmarshalJSONDateTime(t *testing.T) {
	type tm struct {
		DateTime Time `json:"time"`
	}

	data := []struct {
		in, out string
	}{
		{`{"time":180213}`, "180213.000000"},
		{`{"time":180213.13}`, "180213.130000"},
	}

	for i, l := 0, len(data); i < l; i++ {
		dt := tm{}

		err := json.Unmarshal([]byte(data[i].in), &dt)

		if err != nil {
			t.Error(err)
		}

		if f := dt.DateTime.Format(DateFloatFormat); f != data[i].out {
			t.Error("not equal: " + data[i].out + " <=> " + f)
		}
	}
}

func Test_parseDelta(t *testing.T) {
	data := []struct {
		in  string
		dlt Delta
	}{
		{"-0.01", Delta{Duration: -1 * time.Hour, Year: 0, Month: 0, Day: 0}},
		{"-99.01", Delta{Duration: -1 * time.Hour, Year: 0, Month: 0, Day: -99}},
		{"-1099.05", Delta{Duration: -5 * time.Hour, Year: 0, Month: -10, Day: -99}},
		{"-051099.05", Delta{Duration: -5 * time.Hour, Year: -5, Month: -10, Day: -99}},
		{"180213", Delta{Duration: 0, Year: 18, Month: 2, Day: 13}},
		{"180213.000001", Delta{Duration: 1 * time.Second, Year: 18, Month: 2, Day: 13}},
		{"180213.100001", Delta{Duration: 10*time.Hour + 1*time.Second, Year: 18, Month: 2, Day: 13}},
		{"180213.1310", Delta{Duration: 13*time.Hour + 10*time.Minute, Year: 18, Month: 2, Day: 13}},
		{"180213.13", Delta{Duration: 13 * time.Hour, Year: 18, Month: 2, Day: 13}},
		//{"180213.0000004", Delta{Duration: 4 * time.Millisecond, Year: 18, Month: 2, Day: 13}},
		//{"180213.00000033", Delta{Duration: 33 * time.Millisecond, Year: 18, Month: 2, Day: 13}},
	}

	for i, l := 0, len(data); i < l; i++ {
		dt, err := parseDelta([]byte(data[i].in))

		if err != nil {
			t.Error(err)
		}

		if dt != data[i].dlt {
			t.Errorf("not equal (%s) %#v <=> %#v\n", data[i].in, data[i].dlt, dt)
		}
	}
}

func Test_UnmmarshalJSONDelta(t *testing.T) {
	type tm struct {
		Delta Delta `json:"delta"`
	}

	data := []struct {
		in  string
		dlt Delta
	}{
		{`{"delta":180213}`, Delta{Duration: 0, Year: 18, Month: 2, Day: 13}},
		{`{"delta":180213.13}`, Delta{Duration: time.Hour * 13, Year: 18, Month: 2, Day: 13}},
	}

	for i, l := 0, len(data); i < l; i++ {
		dt := tm{}

		err := json.Unmarshal([]byte(data[i].in), &dt)

		if err != nil {
			t.Error(err)
		}

		if dt.Delta != data[i].dlt {
			t.Errorf("not equal %#v <=> %#v\n", data[i].dlt, dt)
		}
	}
}

func Test_handleTimeNow(t *testing.T) {
	r, err := http.NewRequest("GET", "/time/now", nil)

	if err != nil {
		t.Error(err)
	}

	resp := timeNow(r).(Response)

	if resp.Time == nil {
		t.Error("expect time object")
	}
}

func Test_handleTimeAdd(t *testing.T) {
	data := []struct {
		in, out string
	}{
		{"time=180101&delta=99", `{"time":"180410.000000"}`},
		{"time=180101&delta=99.99", `{"time":"180414.030000"}`},
		{"time=100101.12&delta=67.9912", `{"time":"100313.151200"}`},
		{"time=100101.123300123&delta=-9567.9912", `{"time":"011122.092100"}`},
		{"time=180231&delta=99", `{"error":{"message":"parsing time \"180231.000000\": day out of range"}}`},
	}

	for i, l := 0, len(data); i < l; i++ {
		r, err := http.NewRequest("GET", "/time/add?"+data[i].in, nil)

		if err != nil {
			t.Error(err)
		}

		resp := timeAdd(r).(Response)

		var body []byte
		body, err = resp.Get()

		if err != nil {
			t.Error(err)
		}

		if string(body) != data[i].out {
			t.Errorf("not equal %s <=> %s\n", body, data[i].out)
		}
	}
}

func Test_handleTimeSetNoError(t *testing.T) {
	data := []struct {
		in, out string
	}{
		{"time=180101", `{"time":"180410.000000"}`},
		{"time=200101", `{"time":"180414.030000"}`},
		{"time=100101.12", `{"time":"100313.151200"}`},
		{"time=100101.123300123", `{"time":"011122.092100"}`},
	}

	for i, l := 0, len(data); i < l; i++ {
		r, err := http.NewRequest("POST", "/time/set", strings.NewReader(data[i].in))
		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		if err != nil {
			t.Error(err)
		}

		resp := timeSet(r).(Response)

		_, err = resp.Get()

		if err != nil {
			t.Error(err)
		}

		if GlobalDelta.Nanoseconds() == 0 {
			t.Error("emty delta")
		}

		timeRest(r)

		if GlobalDelta.Nanoseconds() != 0 {
			t.Error("delta must be empty")
		}
	}
}
