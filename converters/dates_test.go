package converters

import (
	"testing"
	"time"
)

var testDates = []struct {
	String string    // To be given to Oracle
	Time   time.Time // Golang date
	Binary []byte    // Oracle binary format
}{
	{"DATE '2020-12-31'", time.Date(2020, 12, 31, 0, 0, 0, 0, time.Local), []byte{228, 7, 12, 31, 0, 0, 0, 0}},
}

func TestDecodeDate(t *testing.T) {

	for _, tt := range testDates {
		t.Run(tt.String, func(t *testing.T) {
			got, err := DecodeDate(tt.Binary)
			if err != nil {
				t.Errorf("Unexpected error DecodeDate() error = %v", err)
				return
			}
			if got != tt.Time {
				t.Errorf("Decode(%v)=%v, expected %v", tt.Binary, got, tt.Time)
			}
		})
	}
}
