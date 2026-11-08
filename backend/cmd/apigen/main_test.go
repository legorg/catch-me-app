package main

import "testing"

func TestGeneratedDocumentIsCurrent(t *testing.T) {
	t.Chdir("../..")
	if err := generate(true); err != nil {
		t.Fatal(err)
	}
}
