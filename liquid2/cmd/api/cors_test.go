package main

import (
	"reflect"
	"testing"
)

func TestSplitCommaListTrimsEmptyValues(t *testing.T) {
	got := splitCommaList(" http://localhost:3000, ,http://127.0.0.1:3000 ")
	want := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}
