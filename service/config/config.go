// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright 2024 Canonical Ltd.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package config

import (
	"errors"
	"net"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/canonical/fetch-service/logger"
)

type ACLPolicy int

const (
	Allow ACLPolicy = iota
	Deny
)

func (t ACLPolicy) MarshalYAML() (interface{}, error) {
	switch t {
	case Allow:
		return "allow", nil
	case Deny:
		return "deny", nil
	default:
		return nil, errors.New("invalid ACL policy")
	}
}

func (t *ACLPolicy) UnmarshalYAML(unmarshal func(v interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	switch string(s) {
	case "allow", `"allow"`:
		*t = Allow
		return nil
	case "deny", `"deny"`:
		*t = Deny
		return nil
	default:
		return errors.New("invalid ACL policy")
	}

}

func (t ACLPolicy) String() string {
	switch t {
	case Allow:
		return "allow"
	case Deny:
		return "deny"
	default:
		return ""
	}
}

type IPNet struct {
	net.IPNet
}

func (t IPNet) MarshalYAML() (interface{}, error) {
	return t.String(), nil
}

func (t *IPNet) UnmarshalYAML(unmarshal func(v interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	// Ensure CIDR format.
	if strings.Contains(s, ".") && !strings.Contains(s, "/") {
		s += "/32"
	} else if strings.Contains(s, ":") && !strings.Contains(s, "/") {
		s += "/128"
	}

	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return err
	}
	*t = IPNet{*ipnet}
	return nil
}

type Rule struct {
	Dst    []IPNet   `yaml:"dst"`
	Access ACLPolicy `yaml:"access"`
}

type HttpProxyConfig struct {
	Policy ACLPolicy `yaml:"policy"`
	Rules  []Rule    `yaml:"rules"`
}

type Config struct {
	HttpProxy HttpProxyConfig `yaml:"http-proxy"`
}

var (
	globalConfig     Config
	globalConfigLock sync.Mutex
)

func GetHttpProxyConfig() HttpProxyConfig {
	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()

	cfg := HttpProxyConfig{
		Policy: globalConfig.HttpProxy.Policy,
		Rules:  make([]Rule, len(globalConfig.HttpProxy.Rules)),
	}

	copy(cfg.Rules, globalConfig.HttpProxy.Rules)

	return cfg
}

func SetHttpProxyConfig(cfg HttpProxyConfig) {
	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()

	globalConfig.HttpProxy.Policy = cfg.Policy
	globalConfig.HttpProxy.Rules = make([]Rule, len(cfg.Rules))
	copy(globalConfig.HttpProxy.Rules, cfg.Rules)
}

func LoadHttpProxyRules(filepath string) error {
	if filepath == "" {
		logger.Warningf("Cannot load proxy rules, configuration file not specified.")
		return nil
	}

	logger.Infof("Load proxy rules from %s", filepath)

	f, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	var cfg Config
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return err
	}

	// The configuration is only updated if the configuration file
	// is correctly parsed.
	SetHttpProxyConfig(cfg.HttpProxy)

	logger.Infof("Proxy configuration updated: %d dst rules, default policy: %s",
		len(cfg.HttpProxy.Rules), cfg.HttpProxy.Policy.String())

	return nil
}
