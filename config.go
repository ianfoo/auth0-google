package main

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const (
	DefaultConfigFile = "config"
)

type (
	Config     map[string]UserConfig
	UserConfig struct {
		AccessToken  string
		ExpiresAt    time.Time
		RefreshToken string
	}
)

// Valid is true if the user config has a refresh token.
func (uc UserConfig) Valid() bool {
	return uc.RefreshToken != ""
}

// ValidAccessToken is true if the user config has an access token that has not
// yet expired.
func (uc UserConfig) ValidAccessToken() bool {
	return uc.AccessToken != "" && uc.ExpiresAt.After(time.Now())
}

func SaveConfig(filename string, c Config) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
		return errors.Wrapf(err, "make config dir %s", dir)
	}
	f, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		os.FileMode(0600))
	if err != nil {
		return errors.Wrap(err, "save config file")
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	return enc.Encode(c)
}

func LoadConfig(filename string) (Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return Config{}, errors.Wrap(err, "open config file")
	}
	defer f.Close()
	var (
		dec = gob.NewDecoder(f)
		cfg Config
	)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, errors.Wrap(err, "read config file")
	}
	return cfg, nil
}

func GetConfigDir() (string, error) {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "gaproxy"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "cannot get home directory")
	}
	return filepath.Join(home, ".config", "gaproxy"), nil
}
