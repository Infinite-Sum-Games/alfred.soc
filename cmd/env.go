package cmd

import (
	"fmt"
	"net"
	"strings"

	v "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/spf13/viper"
)

var AppConfig *EnvConfig

type EnvConfig struct {
	Environment   string
	ServerHost    string
	ServerPort    int
	ValkeyHost    string
	ValkeyPort    int
	DatabaseURL   string
	WebhookSecret string
}

// isValidHost must satisfy the following interface to be accepted as a
// validator by ozzo-validation library's validator.By(RuleFunc) method
// func RuleFunc (value any) error {}
func isValidHost(value any) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}
	if strings.ToLower(s) == "localhost" {
		return nil
	}
	if ip := net.ParseIP(s); ip == nil {
		return nil
	}
	if err := is.URL.Validate(s); err == nil {
		return nil
	}
	return fmt.Errorf("must be 'localhost' or a valid URL/IP address")
}

func (e *EnvConfig) Validate() error {
	return v.ValidateStruct(e,
		v.Field(&e.Environment, v.Required, v.In("development", "production")),
		v.Field(&e.ServerHost, v.Required, v.By(isValidHost)),
		v.Field(&e.ServerPort, v.Required, v.Min(1), v.Max(65535)),
		v.Field(&e.ValkeyHost, v.Required, v.By(isValidHost)),
		v.Field(&e.ValkeyPort, v.Required, v.Min(1), v.Max(65535)),
		v.Field(&e.DatabaseURL, v.Required, is.URL),
		v.Field(&e.WebhookSecret, v.Required),
	)
}

func SetupEnv() error {

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	AppConfig = &EnvConfig{
		Environment:   viper.GetString("server.environment"),
		ServerHost:    viper.GetString("server.host"),
		ServerPort:    viper.GetInt("server.port"),
		ValkeyHost:    viper.GetString("valkey.host"),
		ValkeyPort:    viper.GetInt("valkey.port"),
		DatabaseURL:   viper.GetString("database.url"),
		WebhookSecret: viper.GetString("github.webhook_secret"),
	}
	if err := AppConfig.Validate(); err != nil {
		return err
	}
	return nil
}
