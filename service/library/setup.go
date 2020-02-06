package library

import (
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/fubarhouse/pygmy-go/service/dnsmasq"
	"github.com/fubarhouse/pygmy-go/service/haproxy"
	model "github.com/fubarhouse/pygmy-go/service/interface"
	"github.com/fubarhouse/pygmy-go/service/mailhog"
	"github.com/fubarhouse/pygmy-go/service/network"
	"github.com/fubarhouse/pygmy-go/service/resolv"
	"github.com/fubarhouse/pygmy-go/service/ssh/agent"
	"github.com/fubarhouse/pygmy-go/service/ssh/key"
	"github.com/spf13/viper"
)

// Setup holds the core of configuration management with Pygmy.
// It will merge in all the configurations and provide defaults.
func Setup(c *Config) {

	viper.SetDefault("defaults", true)

	var ResolvMacOS = resolv.Resolv{
		Data:    "# Generated by amazeeio pygmy\nnameserver 127.0.0.1\nport 6053\n",
		Enabled: true,
		File:    "docker.amazee.io",
		Folder:  "/etc/resolver",
		Name:    "MacOS Resolver",
	}

	var ResolvGeneric = resolv.Resolv{
		Data:    "nameserver 127.0.0.1 # added by amazee.io pygmy",
		Enabled: true,
		File:    "resolv.conf",
		Folder:  "/etc",
		Name:    "Linux Resolver",
	}

	if runtime.GOOS == "darwin" {
		viper.SetDefault("resolvers", []resolv.Resolv{
			ResolvMacOS,
		})
	} else if runtime.GOOS == "linux" {
		viper.SetDefault("resolvers", []resolv.Resolv{
			ResolvGeneric,
		})
	} else if runtime.GOOS == "windows" {
		viper.SetDefault("resolvers", []resolv.Resolv{})
	}

	e := viper.Unmarshal(&c)

	if e != nil {
		fmt.Println(e)
	}

	if c.Defaults {
		// If Services have been provided in complete or partially,
		// this will override the defaults allowing any value to
		// be changed by the user in the configuration file ~/.pygmy.yml
		if c.Services == nil {
			c.Services = make(map[string]model.Service, 6)
		}
		c.Services["amazeeio-ssh-agent-show-keys"] = getService(key.NewShower(), c.Services["amazeeio-ssh-agent-show-keys"])
		c.Services["amazeeio-ssh-agent-add-key"] = getService(key.NewAdder(), c.Services["amazeeio-ssh-agent-add-key"])
		c.Services["amazeeio-dnsmasq"] = getService(dnsmasq.New(), c.Services["amazeeio-dnsmasq"])
		c.Services["amazeeio-haproxy"] = getService(haproxy.New(), c.Services["amazeeio-haproxy"])
		c.Services["amazeeio-mailhog"] = getService(mailhog.New(), c.Services["amazeeio-mailhog"])
		c.Services["amazeeio-ssh-agent"] = getService(agent.New(), c.Services["amazeeio-ssh-agent"])

		// We need Port 80 to be configured by default.
		// If a port on amazeeio-haproxy isn't explicitly declared,
		// then we should set this value. This is far more creative
		// than needed, so feel free to revisit if you can compile it.
		if c.Services["amazeeio-haproxy"].HostConfig.PortBindings == nil {
			c.Services["amazeeio-haproxy"] = getService(haproxy.NewDefaultPorts(), c.Services["amazeeio-haproxy"])
		}

		// It's sensible to use the same logic for port 1025.
		// If a user needs to configure it, the default value should not be set also.
		if c.Services["amazeeio-mailhog"].HostConfig.PortBindings == nil {
			c.Services["amazeeio-mailhog"] = getService(mailhog.NewDefaultPorts(), c.Services["amazeeio-mailhog"])
		}

		// Ensure Networks has a at least a zero value.
		// We should provide defaults for amazeeio-network when no value is provided.
		if c.Networks == nil {
			c.Networks = make(map[string]types.NetworkResource, 0)
			c.Networks["amazeeio-network"] = getNetwork(network.New(), c.Networks["amazeeio-network"])
		}
		
		// Ensure Volumes has a at least a zero value.
		if c.Volumes == nil {
			c.Volumes = make(map[string]types.Volume, 0)
		}

		for _, v := range c.Volumes {
			// Get the potentially existing volume:
			c.Volumes[v.Name], _ = model.DockerVolumeGet(v.Name)
			// Merge the volume with the provided configuration:
			c.Volumes[v.Name] = getVolume(c.Volumes[v.Name], c.Volumes[v.Name])
		}
	}

	c.SortedServices = make([]string, 0, len(c.Services))
	for key, service := range c.Services {
		weight, _ := service.GetFieldInt("weight")
		c.SortedServices = append(c.SortedServices, fmt.Sprintf("%v|%v", weight, key))
	}
	sort.Strings(c.SortedServices)

	for n, v := range c.SortedServices {
		c.SortedServices[n] = strings.Split(v, "|")[1]
	}

}
