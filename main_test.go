package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func check(t *testing.T, e error, s string) {
	if e != nil {
		t.Errorf("%s - %v", s, e)
	}
}

func TestFindConfigFile(t *testing.T) {
	_, err := findConfigFile()

	if err == nil {
		t.Error("No error on config file not found")
	}

	f, err := os.CreateTemp("", "dex-test")
	check(t, err, "Error creating cfg file")

	defer os.Remove(f.Name())

	f2, err := os.CreateTemp("", "dex-test")
	check(t, err, "Error creating second cfg file")

	defer os.Remove(f2.Name())

	configFileLocations = []string{"not-exists.yml", f.Name(), f2.Name()}

	cfg, err := findConfigFile()
	check(t, err, "config file not found")

	assert.Equal(t, cfg, f.Name())

	os.Remove(f.Name())

	cfg2, err := findConfigFile()
	check(t, err, "config file not found")

	assert.Equal(t, cfg2, f2.Name())

	f3, err := os.CreateTemp("", "dex-test")
	check(t, err, "Error creating second cfg file")

	os.Setenv("DEX_FILE", f3.Name())

	cfg3, err := findConfigFile()
	check(t, err, "config file not found")

	assert.Equal(t, cfg3, f3.Name())

}
