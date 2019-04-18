package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ianfoo/auth0-provider-identity/tmplsrv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	port            = flag.Uint("port", 3000, "Port to listen on")
	apiClientID     = flag.String("api-client-id", "", "Client ID for Auth0 API access")
	apiClientSecret = flag.String("client-secret", "", "Client secret for Auth0 API access")
	appClientID     = flag.String("app-client-id", "", "Client ID for client app")
	auth0Domain     = flag.String("auth0-domain", "", "Auth0 domain (e.g., yourdomain.auth0.com)")
	callbackURL     = flag.String("callback-url", "http://localhost:3000", "OAuth2 callback URL")
	staticDir       = flag.String(
		"static-dir",
		filepath.Join(filepath.Clean(Must(os.Getwd())), "static"),
		"Directory from which static content is served")
	templateDir = flag.String(
		"template-dir",
		filepath.Join(filepath.Clean(Must(os.Getwd())), "templates"),
		"Directory from which templated content is served")

	// TODO This is temporary: the user ID will be provided by the SPA login process,
	// which will call the server with the User ID after handling the callback from Auth0.
	userID = flag.String("user-id", "", "Auth0 User ID for whom to fetch user details")

	configFile = flag.String("config", "", "Use specified config instead of default")
	readConfig = flag.Bool("read-config", false, "Dump config (if available) and exit")
	verbose    = flag.Bool("verbose", false, "Log more effusively")
)

func main() {
	flag.Parse()
	if *verbose {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.WithField("verbose", true).Info("verbose logging enabled")
	}
	auth0Config := Auth0Config{
		ClientID:     getFromFlagOrEnv(*apiClientID, "API_CLIENT_ID", "API client ID"),
		ClientSecret: getFromFlagOrEnv(*apiClientSecret, "API_CLIENT_SECRET", "API client secret"),
		Domain:       getFromFlagOrEnv(*auth0Domain, "AUTH0_DOMAIN", "Auth0 domain"),
	}
	bail(GetAndUpdateConfig(getFromFlagOrEnv(*userID, "USER_ID", "User ID"), auth0Config))

	setUpHandlers(map[string]interface{}{
		"ClientID":    getFromFlagOrEnv(*appClientID, "APP_CLIENT_ID", "App client ID"),
		"CallbackURL": getFromFlagOrEnv(*callbackURL, "CALLBACK_URL", "Callback URL"),
		"Domain":      auth0Config.Domain,
	})

	logrus.WithFields(logrus.Fields{
		"static_dir":   *staticDir,
		"template_dir": *templateDir,
		"port":         *port,
	}).Info("server listening")
	bail(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

func GetAndUpdateConfig(userID string, auth0Config Auth0Config) error {
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
		LogUserConfig(userID, uc)
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
	LogUserConfig(userID, uc)

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
func setUpHandlers(tmplData map[string]interface{}) {
	http.Handle("/", tmplsrv.TemplateServer(
		http.Dir(*staticDir),
		http.Dir(*templateDir),
		tmplData))
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
		userID := pathSlice[len(pathSlice)-1]
		logrus.WithFields(logrus.Fields{
			"user_id": userID,
		}).Info("getting user config")
		rw.Write([]byte(fmt.Sprint("User ID is %q", userID)))
	})
	http.HandleFunc("/favicon.ico", favicon404)

	// logrus.WithFields(logrus.Fields{
	//     "client_id": auth0Cfg.ClientID,
	//     "auth0_url": auth0Cfg.Domain,
	//     "static_dir": *staticDir,
	//     "template_dir": *templateDir,
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

func LogUserConfig(userID string, uc UserConfig) {
	logrus.WithFields(logrus.Fields{
		"user_id":       userID,
		"access_token":  uc.AccessToken,
		"expires_at":    uc.ExpiresAt,
		"refresh_token": uc.RefreshToken,
	}).Info("user configuration")
}

// favicon handler was taking over half a second (wha? requires profiling)
// so just respond with a 404.
func favicon404(rw http.ResponseWriter, _ *http.Request) {
	http.Error(rw, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func favicon(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "image/x-icon")
	rw.Header().Set("Cache-Control", "public, max-age=7776000")
	fmt.Fprintln(rw, "data:image/x-icon;base64,iVBORw0KGgoAAAANSUhEUgAAA"+
		"BAAAAAQEAYAAABPYyMiAAAABmJLR0T///////8JWPfcAAAACXB"+
		"IWXMAAABIAAAASABGyWs+AAAAF0lEQVRIx2NgGAWjYBSMglEwC"+
		"kbBSAcACBAAAeaR9cIAAAAASUVORK5CYII=\n")
}
