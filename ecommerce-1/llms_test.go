package main

import (
	"testing"
)

func TestInitialiseLLMClients(t *testing.T) {
	localServerUrl := "http://localhost:11434/v1"
	clients := initialiseLLMClients(localServerUrl)

	// Check that the number of clients is non-zero
	if len(clients) == 0 {
		t.Errorf("No clients were initialized, expected at least one client")
	}
}
