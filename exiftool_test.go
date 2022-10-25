package exiftool

import (
	"testing"
)

func TestExiftool(t *testing.T) {
	e, err := NewExiftool()
	defer e.Close()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{"../testdata/gps.jpg", "../testdata/binary.mp3", "../testdata/binary.mp3", "../testdata/binary.mp3", "../testdata/binary.mp3", "../testdata/binary.mp3", "../testdata/binary.mp3"} {
		out, err := e.Scan(f)
		if err != nil {
			t.Log(err)
			continue
		}
		t.Log(out)
	}

}
