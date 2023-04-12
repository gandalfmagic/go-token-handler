package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	defaultIsProduction              = false
	defaultLogLevel                  = "info"
	defaultOidcIssuer                = ""
	defaultOidcClientID              = ""
	defaultOidcClientSecret          = ""
	defaultOidcRedirectURL           = ""
	defaultOidcPostLoginRedirectURL  = ""
	defaultOidcPostLogoutRedirectURL = ""
	defaultListenAddr                = ":9080"
	defaultCookieDomain              = "localhost"
	defaultCookieName                = "session"
	defaultSessionAuthSecret         = "my-secret-key-CHANGE-ME-IN-PROD!"
	defaultSessionEncSecret          = ""
	defaultSessionOldAuthSecret      = ""
	defaultSessionOldEncSecret       = ""
	defaultSessionDBKey              = ""
	defaultSessionOldDBKey           = ""
	defaultProxyConfig               = ""
	defaultDBType                    = "sqlite"
	defaultDBHost                    = ""
	defaultDBName                    = ""
	defaultDBUsername                = ""
	defaultDBPassword                = ""
)

var (
	ErrMissingParameter                = errors.New("a mandatory parameter is missing")
	ErrDefaultCookieDomainInProduction = errors.New("the cookie domain shouldn't be 'localhost' in production")
	ErrDefaultAuthSecretInProduction   = errors.New("the session authentication secret should be changed to a random value in production")
	ErrWrongAuthSecretSize             = errors.New("the session authentication secret should have a size of 32 or 64 bytes")
	ErrWrongEncSecretSize              = errors.New("the session encryption secret should have a size of 16, 24 or 32 bytes")
	ErrWrongEncDBKey                   = errors.New("the database encryption key must have a size of 32 bytes")
	ErrWrongDBType                     = errors.New("the database backend must be a value from: sqlite, postgresql")
	ErrMissingSQLiteDatabase           = errors.New("you must specify the database file name, using the db-name parameter")
	ErrMissingDBServerHost             = errors.New("you must specify the database host, using the db-host parameter")
	ErrMissingDBServerDatabase         = errors.New("you must specify the database name, using the db-name parameter")
	ErrMissingDBServerUsername         = errors.New("you must specify the username to connect to the database, using the db-username parameter")
	ErrMissingDBServerPassword         = errors.New("you must specify the password to connect to the database, using the db-password parameter")
)

// Config stores all then configuration of the application.
// The values are read by Viper from a configuration file or from environment variables.
type Config struct {
	IsProduction              bool   `mapstructure:"IS_PRODUCTION"`
	LogLevel                  string `mapstructure:"LOG_LEVEL"`
	OidcIssuer                string `mapstructure:"OIDC_ISSUER"`
	OidcClientID              string `mapstructure:"OIDC_CLIENT_ID"`
	OidcClientSecret          string `mapstructure:"OIDC_CLIENT_SECRET"`
	OidcRedirectURL           string `mapstructure:"OIDC_REDIRECT_URL"`
	OidcPostLoginRedirectURL  string `mapstructure:"OIDC_POST_LOGIN_REDIRECT_URL"`
	OidcPostLogoutRedirectURL string `mapstructure:"OIDC_POST_LOGOUT_REDIRECT_URL"`
	ListenAddr                string `mapstructure:"LISTEN_ADDR"`
	CookieDomain              string `mapstructure:"COOKIE_DOMAIN"`
	CookieName                string `mapstructure:"COOKIE_NAME"`
	SessionAuthSecret         string `mapstructure:"SESSION_AUTH_SECRET"`
	SessionEncSecret          string `mapstructure:"SESSION_ENC_SECRET"`
	SessionOldAuthSecret      string `mapstructure:"SESSION_OLD_AUTH_SECRET"`
	SessionOldEncSecret       string `mapstructure:"SESSION_OLD_ENC_SECRET"`
	SessionDBKey              string `mapstructure:"SESSION_DB_KEY"`
	SessionOldDBKey           string `mapstructure:"SESSION_OLD_DB_KEY"`
	ProxyConfig               string `mapstructure:"PROXY_CONFIG"`
	DBType                    string `mapstructure:"DB_TYPE"`
	DBHost                    string `mapstructure:"DB_HOST"`
	DBName                    string `mapstructure:"DB_NAME"`
	DBUsername                string `mapstructure:"DB_USERNAME"`
	DBPassword                string `mapstructure:"DB_PASSWORD"`
}

// LoadConfig reads the configuration from a file or from environment variables.
func LoadConfig() (config Config, err error) {
	viper.SetDefault("IS_PRODUCTION", defaultIsProduction)
	viper.SetDefault("LOG_LEVEL", defaultLogLevel)
	viper.SetDefault("OIDC_ISSUER", defaultOidcIssuer)
	viper.SetDefault("OIDC_CLIENT_ID", defaultOidcClientID)
	viper.SetDefault("OIDC_CLIENT_SECRET", defaultOidcClientSecret)
	viper.SetDefault("OIDC_REDIRECT_URL", defaultOidcRedirectURL)
	viper.SetDefault("OIDC_POST_LOGIN_REDIRECT_URL", defaultOidcPostLoginRedirectURL)
	viper.SetDefault("OIDC_POST_LOGOUT_REDIRECT_URL", defaultOidcPostLogoutRedirectURL)
	viper.SetDefault("LISTEN_ADDR", defaultListenAddr)
	viper.SetDefault("COOKIE_DOMAIN", defaultCookieDomain)
	viper.SetDefault("COOKIE_NAME", defaultCookieName)
	viper.SetDefault("SESSION_AUTH_SECRET", defaultSessionAuthSecret)
	viper.SetDefault("SESSION_ENC_SECRET", defaultSessionEncSecret)
	viper.SetDefault("SESSION_OLD_AUTH_SECRET", defaultSessionOldAuthSecret)
	viper.SetDefault("SESSION_OLD_ENC_SECRET", defaultSessionOldEncSecret)
	viper.SetDefault("SESSION_DB_KEY", defaultSessionDBKey)
	viper.SetDefault("SESSION_OLD_DB_KEY", defaultSessionOldDBKey)
	viper.SetDefault("PROXY_CONFIG", defaultProxyConfig)
	viper.SetDefault("DB_TYPE", defaultDBType)
	viper.SetDefault("DB_HOST", defaultDBHost)
	viper.SetDefault("DB_NAME", defaultDBName)
	viper.SetDefault("DB_USERNAME", defaultDBUsername)
	viper.SetDefault("DB_PASSWORD", defaultDBPassword)
	viper.AutomaticEnv()

	flag.Bool("is-production", defaultIsProduction, "configure for a production environment")
	zap.LevelFlag("log-level", zap.InfoLevel, "set the logging level")
	flag.String("oidc-issuer", defaultOidcIssuer, "the url of the oidc issuer")
	flag.String("oidc-client-id", defaultOidcClientID, "the oidc client-id")
	flag.String("oidc-client-secret", defaultOidcClientSecret, "the oidc client-secret")
	flag.String("oidc-redirect-url", defaultOidcRedirectURL, "the endpoint where to mount the oidc login callback")
	flag.String("oidc-post-login-redirect-url", defaultOidcPostLoginRedirectURL, "where to redirect the client after a valid login")
	flag.String("oidc-post-logout-redirect-url", defaultOidcPostLogoutRedirectURL, "where to redirect the client after a logout")
	flag.String("listen-addr", defaultListenAddr, "define the address where the main service will listen on")
	flag.String("cookie-domain", defaultCookieDomain, "the domain for the session cookie")
	flag.String("cookie-name", defaultCookieName, "the name of the session cookie")
	flag.String("session-auth-secret", defaultSessionAuthSecret, "the authentication key for the session cookie")
	flag.String("session-enc-secret", defaultSessionEncSecret, "the encryption key for the session cookie")
	flag.String("session-old-auth-secret", defaultSessionOldAuthSecret, "the old authentication key for the session cookie (rotation)")
	flag.String("session-old-enc-secret", defaultSessionOldEncSecret, "the old encryption key for the session cookie (rotation)")
	flag.String("session-db-key", defaultSessionDBKey, "the encryption key for the session db storage")
	flag.String("session-old-db-key", defaultSessionOldDBKey, "the old encryption key for the session db storage (rotation)")
	flag.String("proxy-config", defaultProxyConfig, "the path to the proxy configuration file")
	flag.String("db-type", defaultDBType, "the database backend used (postgresql, sqlite)")
	flag.String("db-host", defaultDBHost, "the database server hostname or ip address")
	flag.String("db-name", defaultDBName, "the database name")
	flag.String("db-username", defaultDBUsername, "the username to use to connect the database")
	flag.String("db-password", defaultDBPassword, "the password to use to connect the database")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	if err = viper.BindPFlags(pflag.CommandLine); err != nil {
		return
	}

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.token_handler")
	viper.AddConfigPath("/etc/token_handler")
	viper.SetConfigName("token_handler")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig()

	if err = viper.Unmarshal(&config); err != nil {
		return
	}

	return config.configValidate()
}

func (c Config) configValidate() (Config, error) {
	if c.OidcIssuer == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "oidc-issuer")
	}

	if c.OidcClientID == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "oidc-client-id")
	}

	if c.OidcClientSecret == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "oidc-client-secret")
	}

	if c.OidcRedirectURL == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "oidc-redirect-url")
	}

	if c.OidcPostLoginRedirectURL == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "oidc-post-login-redirect-url")
	}

	if c.OidcPostLogoutRedirectURL == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "oidc-post-logout-redirect-url")
	}

	if c.ListenAddr == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "listen-addr")
	}

	if c.CookieDomain == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "cookie-domain")
	}

	if c.CookieName == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "cookie-name")
	}

	if c.SessionAuthSecret == "" {
		return c, fmt.Errorf("%w: %s", ErrMissingParameter, "session-secret")
	}

	if c.CookieDomain == defaultCookieDomain && c.IsProduction {
		return c, ErrDefaultCookieDomainInProduction
	}

	if c.SessionAuthSecret == defaultSessionAuthSecret && c.IsProduction {
		return c, ErrDefaultAuthSecretInProduction
	}

	// It is recommended to use an authentication key with 32 or 64 bytes.
	lenSessionAuthSecret := len(c.SessionAuthSecret)
	if lenSessionAuthSecret != 32 && lenSessionAuthSecret != 64 {
		return c, fmt.Errorf("%w: %s", ErrWrongAuthSecretSize, "session-auth-secret")
	}

	// The encryption key, if set, must be either 16, 24, or 32 bytes to select
	lenSessionEncSecret := len(c.SessionEncSecret)
	if c.SessionEncSecret != "" && lenSessionEncSecret != 16 && lenSessionEncSecret != 24 && lenSessionEncSecret != 32 {
		return c, fmt.Errorf("%w: %s", ErrWrongEncSecretSize, "session-enc-secret")
	}

	// It is recommended to use an authentication key with 32 or 64 bytes (if the old auth key is used).
	lenSessionOldAuthSecret := len(c.SessionOldAuthSecret)
	if c.SessionOldAuthSecret != "" && lenSessionOldAuthSecret != 32 && lenSessionOldAuthSecret != 64 {
		return c, fmt.Errorf("%w: %s", ErrWrongAuthSecretSize, "session-old-auth-secret")
	}

	// The encryption key, if set, must be either 16, 24, or 32 bytes to select
	lenSessionOldEncSecret := len(c.SessionOldEncSecret)
	if c.SessionOldEncSecret != "" && lenSessionOldEncSecret != 16 && lenSessionOldEncSecret != 24 && lenSessionOldEncSecret != 32 {
		return c, fmt.Errorf("%w: %s", ErrWrongEncSecretSize, "session-old-enc-secret")
	}

	// The encryption key, if set, must be 32 bytes
	lenSessionDBKey := len(c.SessionDBKey)
	if c.SessionDBKey != "" && lenSessionDBKey != 32 {
		return c, fmt.Errorf("%w: %s", ErrWrongEncDBKey, "session-db-key")
	}

	// The encryption key, if set, must be 32 bytes
	lenSessionOldDBKey := len(c.SessionOldDBKey)
	if c.SessionOldDBKey != "" && lenSessionOldDBKey != 32 {
		return c, fmt.Errorf("%w: %s", ErrWrongEncDBKey, "session-old-db-key")
	}

	// The database backend must be supported
	if c.DBType != "sqlite" && c.DBType != "postgresql" {
		return c, ErrWrongDBType
	}

	// For SQLite the db name must be populated (used as filename for the db)
	if c.DBType == "sqlite" && c.DBName == "" {
		return c, ErrMissingSQLiteDatabase
	}

	// For Postgresql the db host must be populated
	if c.DBType == "postgresql" {
		if c.DBHost == "" {
			return c, ErrMissingDBServerHost
		}

		if c.DBName == "" {
			return c, ErrMissingDBServerDatabase
		}

		if c.DBUsername == "" {
			return c, ErrMissingDBServerUsername
		}

		if c.DBPassword == "" {
			return c, ErrMissingDBServerPassword
		}
	}

	return c, nil
}

type ProxyConfigData struct {
	Proxies []struct {
		Endpoint   string `yaml:"endpoint"`
		Target     string `yaml:"target"`
		Parameters struct {
			IdleConnTimeout time.Duration `yaml:"idleConnTimeout"`
			MaxIdleConns    int           `yaml:"maxIdleConns"`
			DialKeepAlive   time.Duration `yaml:"dialKeepAlive"`
			DialTimeout     time.Duration `yaml:"dialTimeout"`
		} `yaml:"parameters"`
	} `yaml:"proxies"`
}

func (c Config) ReadProxyConfig() (ProxyConfigData, error) {
	pcFile, err := os.ReadFile(c.ProxyConfig)
	if err != nil {
		return ProxyConfigData{}, err
	}

	var data ProxyConfigData

	err = yaml.Unmarshal(pcFile, &data)
	if err != nil {
		return ProxyConfigData{}, err
	}

	return data, nil
}
