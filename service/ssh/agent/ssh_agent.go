package agent

import (
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	model "github.com/fubarhouse/pygmy-go/service/interface"
	"github.com/fubarhouse/pygmy-go/service/ssh/key"
)

// New will provide the standard object for the SSH agent container.
func New() model.Service {
	return model.Service{
		Config: container.Config{
			Image: "amazeeio/ssh-agent",
			Labels: map[string]string{
				"pygmy":         "pygmy",
				"pygmy.name":    "amazeeio-ssh-agent",
				"pygmy.output":  "false",
				"pygmy.purpose": "sshagent",
				"pygmy.weight":  "30",
			},
		},
		HostConfig: container.HostConfig{
			AutoRemove: false,
			RestartPolicy: struct {
				Name              string
				MaximumRetryCount int
			}{Name: "always", MaximumRetryCount: 0},
		},
		NetworkConfig: network.NetworkingConfig{},
	}
}

// List will start the active 'Shower', and return its stdout.
func List() []byte {
	i := key.NewShower()
	r, _ := i.Start()
	return r
}

// Search will determine if an SSH key has been added to the agent.
func Search(key string) bool {
	items := List()
	for _, item := range strings.Split(string(items), "\n") {
		if strings.Contains(item, "The agent has no identities") {
			return false
		}
		if strings.Contains(item, key) {
			return true
		}
	}
	return false
}
