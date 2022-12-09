package exiftool

import (
	"testing"
)

func TestExiftool(t *testing.T) {
	e, err := NewExiftool()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	for _, f := range []string{"./exif.go"} {
		out, err := e.Scan(f)
		if err != nil {
			t.Log(err)
			continue
		}
		t.Log(out)
	}
}
