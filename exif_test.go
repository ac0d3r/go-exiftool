package exiftool

import (
	"context"
	"testing"
	"time"
)

func TestReuseExiftool(t *testing.T) {
	var (
		re  *ReuseExiftool
		err error
	)
	re, err = NewReuseExiftool(context.Background(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"../testdata/gps.jpg", "../testdata/binary.mp3", "../testdata/binary.mp3", "../testdata/binary.mp3", "../testdata/binary.mp3", "../testdata/binary.mp3", "../testdata/binary.mp3"} {
		out, err := re.Scan(f)
		if err != nil {
			t.Error(err)
			continue
		}
		t.Log(out)
	}
}
