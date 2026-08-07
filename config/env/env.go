package env

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func Load() {
	_ = godotenv.Load()
	env := os.Getenv("ENV")
	configName := "config.local"
	configpath := "."
	switch env {
	case "production":
		configName = "config"
	case "test":
		configName = "config.test"
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configpath)
	if err := viper.ReadInConfig(); err != nil {
		panic("failed to read config")
	}
}
