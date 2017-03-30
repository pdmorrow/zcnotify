package main

import (
	"errors"
	"fmt"
	"github.com/badoux/checkmail"
)

type emailConfig struct {
	From     string
	To       string
	Ssl      bool
	Server   string
	Password string
}

type interfaceConfig struct {
	Use     []string
	Exclude []string
	Ip      []string
}

const (
	DEFAULT_SERVICE     string = "_workstation._tcp"
	DEFAULT_DOMAIN      string = "local"
	DEFAULT_SCAN_PERIOD uint   = 10
)

type zeroconfConfig struct {
	Service string
	Domain  string
}

type config struct {
	ScanPeriodSeconds uint
	NotifyTypes       []string
	Zeroconf          zeroconfConfig
	Interfaces        interfaceConfig
	Email             map[string]emailConfig
}

func ValidEmailConfig(emailConfs map[string]emailConfig) error {
	for cfgName, emailConf := range emailConfs {
		if err := checkmail.ValidateFormat(emailConf.From); err != nil {
			return errors.New(fmt.Sprintf("email config: %q from: %q %s",
				cfgName, emailConf.From, err.Error()))
		}

		if err := checkmail.ValidateFormat(emailConf.To); err != nil {
			return errors.New(fmt.Sprintf("email config: %q to: %q %s",
				cfgName, emailConf.To, err.Error()))
		}

		if emailConf.Server == "" {
			return errors.New(fmt.Sprintf("email config: %q no server specified", cfgName))
		}
	}

	return nil
}
