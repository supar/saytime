package main

import (
	"encoding/json"
	"strconv"
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
		{"180213", Delta{Duration: 0, Year: 18, Month: 2, Day: 13}},
		{"180213.000001", Delta{Duration: 1 * time.Second, Year: 18, Month: 2, Day: 13}},
		{"180213.100001", Delta{Duration: 10*time.Hour + 1*time.Second, Year: 18, Month: 2, Day: 13}},
		{"180213.1310", Delta{Duration: 13*time.Hour + 10*time.Minute, Year: 18, Month: 2, Day: 13}},
		{"180213.13", Delta{Duration: 13 * time.Hour, Year: 18, Month: 2, Day: 13}},
		{"180213.0000004", Delta{Duration: 4 * time.Millisecond, Year: 18, Month: 2, Day: 13}},
		{"180213.00000033", Delta{Duration: 33 * time.Millisecond, Year: 18, Month: 2, Day: 13}},
	}

	for i, l := 0, len(data); i < l; i++ {
		dt, err := parseDelta([]byte(data[i].in))

		if err != nil {
			t.Error(err)
		}

		if dt != data[i].dlt {
			t.Errorf("not equal %#v <=> %#v\n", data[i].dlt, dt)
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
