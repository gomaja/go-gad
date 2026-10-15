package gad

import (
	"reflect"
	"testing"
)

func wantEqual[T any](t *testing.T, name string, want, got T) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("%s: got %v, want %v", name, got, want)
	}
}
