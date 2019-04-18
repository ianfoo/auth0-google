package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	port         = flag.Uint("port", 3000, "Port to listen on")
	clientID     = flag.String("client-id", "", "Client ID for Auth0 API access")
	clientSecret = flag.String("client-secret", "", "Client secret for Auth0 API access")
	auth0Domain  = flag.String("auth0-domain", "", "Auth0 domain (e.g., yourdomain.auth0.com)")
	serveDir     = flag.String(
		"serve-dir",
		filepath.Join(filepath.Clean(Must(os.Getwd())), "static"),
		"Directory from which content is served")

	// TODO This is temporary: the user ID will be provided by the SPA login process,
	// which will call the server with the User ID after handling the callback from Auth0.
	userID = flag.String("user-id", "", "Auth0 User ID for whom to fetch user details")

	configFile = flag.String("config", "", "Use specified config instead of default")
	readConfig = flag.Bool("read-config", false, "Dump config (if available) and exit")
)

func main() {
	flag.Parse()
	bail(GetAndUpdateConfig(*userID, Auth0Config{
		ClientID:     getFromFlagOrEnv(*clientID, "CLIENT_ID", "Client ID"),
		ClientSecret: getFromFlagOrEnv(*clientSecret, "CLIENT_SECRET", "Client secret"),
		Domain:       getFromFlagOrEnv(*auth0Domain, "AUTH0_DOMAIN", "Auth0 domain"),
	}))
}

func GetAndUpdateConfig(userID string, auth0Config Auth0Config) error {
	flag.Parse()
	// setUpHandlers()
	// bail(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))

	cfgFile, err := GetConfigFile()
	if err != nil {
		return err
	}
	cfg, err := LoadConfig(cfgFile)
	if err != nil && !os.IsNotExist(errors.Cause(err)) {
		return errors.Wrap(err, "reading config file")
	}
	if uc, ok := cfg[userID]; ok {
		logrus.WithFields(logrus.Fields{
			"user_id":     userID,
			"config_file": cfgFile,
		}).Info("read from config file")
		LogUserConfig(uc)
		return nil
	}

	auth0Client, err := NewAuth0Client(auth0Config)
	if err != nil {
		return err
	}
	uc, err := GetUserConfigFromAuth0(auth0Client, userID)
	if err != nil {
		return err
	}
	LogUserConfig(uc)

	// Update config with the new user details.
	cfg[userID] = uc
	if err := SaveConfig(cfgFile, cfg); err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{"config_file": cfgFile}).Info("saved config")
	return nil
}

func GetUserConfigFromAuth0(auth0Client Auth0Client, userID string) (UserConfig, error) {
	ud, err := auth0Client.GetUserDetails(userID)
	if err != nil {
		return UserConfig{}, err
	}

	provider := strings.Split(userID, "|")[0]
	if provider == userID {
		return UserConfig{}, errors.Errorf("unable to get provider from user ID %q", userID)
	}

	// Find the requested identity.
	id, err := ud.Identity(provider)
	if err != nil {
		return UserConfig{}, err
	}
	logrus.WithFields(logrus.Fields{
		"user_id":  userID,
		"provider": provider,
	}).Info("got user config from Auth0")
	return id.ToUserConfig(), nil
}

// Static server that logs requests, to allow seeing what Auth0 calls back with.
func setUpHandlers() {
	fs := http.FileServer(http.Dir(*serveDir))
	http.Handle("/", fs)
	http.HandleFunc("/login", func(rw http.ResponseWriter, r *http.Request) {
		logrus.WithFields(logrus.Fields{
			"remote_addr": r.RemoteAddr,
		}).Info("handling login")
		rw.Write([]byte("handled login"))
	})
	http.HandleFunc("/user-config", func(rw http.ResponseWriter, r *http.Request) {
		pathSlice := strings.Split(r.URL.Path, "/")
		if len(pathSlice) == 1 {
			http.Error(rw, "no user ID specified", http.StatusBadRequest)
			return
		}
		user := pathSlice[len(pathSlice)-1]
		rw.Write([]byte(fmt.Sprint("User is %q", user)))
	})

	// logrus.WithFields(logrus.Fields{
	//     "client_id": auth0Cfg.ClientID,
	//     "auth0_url": auth0Cfg.Domain,
	//     "serve_dir": *serveDir,
	//     "port":      *port,
	// }).Info("Serving for OAuth authorization")
}

func GetConfigFile() (string, error) {
	if *configFile != "" {
		return *configFile, nil
	}
	cfgDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, DefaultConfigFile), nil
}

func LogUserConfig(uc UserConfig) {
	logrus.WithFields(logrus.Fields{
		"user_id":       userID,
		"access_token":  uc.AccessToken,
		"expires_at":    uc.ExpiresAt,
		"refresh_token": uc.RefreshToken,
	}).Info("user configuration")
}
