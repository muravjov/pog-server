package util

import (
	"github.com/spf13/viper"
)

func bindEnv(name string) {
	_ = viper.BindEnv(name, name)
}

func StringEnv(variable *string, name string, defValue string) {
	bindEnv(name)
	if defValue != "" {
		viper.SetDefault(name, defValue)
	}
	*variable = viper.GetString(name)
}

func BoolEnv(variable *bool, name string, defValue bool) {
	bindEnv(name)
	viper.SetDefault(name, defValue)
	*variable = viper.GetBool(name)
}
