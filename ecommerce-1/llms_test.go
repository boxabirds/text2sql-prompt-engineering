package main

import (
	"testing"
)

func TestGetLLMs(t *testing.T) {
	initializers := getLLMs("http://localhost:11434/v1/")
	if len(initializers) == 0 {
		t.Error("Expected non-zero length slice of initializers")
	}

	for _, initializer := range initializers {
		if initializer.InitFunc == nil {
			t.Errorf("Expected non-nil function for %s", initializer.Name)
		}
		if len(initializer.Name) == 0 {
			t.Errorf("Expected non-zero length name for the initializer, got empty string")
		}
	}
}
