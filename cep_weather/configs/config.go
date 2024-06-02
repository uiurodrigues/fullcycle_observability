package configs

import "github.com/spf13/viper"

var cfg *conf

type conf struct {
	WeatherAPIKey string `mapstructure:"WEATHER_API_KEY"`
}

func GetConfig() *conf {
	if cfg != nil {
		return cfg
	}
	cfg, err := loadConfig(".")
	if err != nil {
		panic(err)
	}
	return cfg
}

func loadConfig(path string) (*conf, error) {
	var cfg *conf
	viper.SetConfigName("app_config")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.AddConfigPath("./app") // <- to work with Dockerfile setup
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		panic(err)
	}

	return cfg, err
}
