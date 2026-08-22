package utils

import (
	"reflect"
	"testing"
)

func TestCleanStrSlice(t *testing.T) {
	input := []string{" first ", "", "  ", "second", "\tthird\n"}
	want := []string{"first", "second", "third"}
	if got := CleanStrSlice(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("CleanStrSlice() = %#v, want %#v", got, want)
	}
	if got := CleanStrSlice(nil); got != nil {
		t.Fatalf("CleanStrSlice(nil) = %#v, want nil", got)
	}
}
