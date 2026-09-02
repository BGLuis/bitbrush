package generators

import (
	"encoding/json"
	"image"
	"testing"
)

func TestRegistryDispatch(t *testing.T) {
	var gotParams string
	Register("fake-test-gen", func(p json.RawMessage, w, h int) (*image.RGBA, error) {
		gotParams = string(p)
		return image.NewRGBA(image.Rect(0, 0, w, h)), nil
	})

	img, err := Render("fake-test-gen", json.RawMessage(`{"k":1}`), 8, 6)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds() != image.Rect(0, 0, 8, 6) {
		t.Fatalf("bad bounds %v", img.Bounds())
	}
	if gotParams != `{"k":1}` {
		t.Fatalf("params not passed through: %q", gotParams)
	}

	found := false
	for _, n := range Names() {
		if n == "fake-test-gen" {
			found = true
		}
	}
	if !found {
		t.Fatal("Names() missing the registered generator")
	}
}

func TestRegistryErrors(t *testing.T) {
	if _, err := Render("nope-not-registered", nil, 4, 4); err == nil {
		t.Fatal("want error for unknown generator")
	}
	Register("size-check-gen", func(p json.RawMessage, w, h int) (*image.RGBA, error) {
		return image.NewRGBA(image.Rect(0, 0, w, h)), nil
	})
	if _, err := Render("size-check-gen", nil, 0, 10); err == nil {
		t.Fatal("want error for non-positive size")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate Register should panic")
		}
	}()
	Register("dup-gen", func(json.RawMessage, int, int) (*image.RGBA, error) { return nil, nil })
	Register("dup-gen", func(json.RawMessage, int, int) (*image.RGBA, error) { return nil, nil })
}
