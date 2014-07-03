// logstash-forwarder config handling
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/digital-wonderland/docker-logstash-forwarder/utils"
	docker "github.com/fsouza/go-dockerclient"
)

// Network section of a logstash-forwarder configuration
type Network struct {
	Servers        []string `json:"servers"`
	SslCertificate string   `json:"ssl certificate"`
	SslKey         string   `json:"ssl key"`
	SslCa          string   `json:"ssl ca"`
	Timeout        int64    `json:"timeout"`
}

// File section of a logstash-forwarder configuration
type File struct {
	Paths  []string          `json:"paths"`
	Fields map[string]string `json:"fields"`
}

// Logstash-forwarder configuration root
type LogstashForwarderConfig struct {
	Network Network `json:"network"`
	Files   []File  `json:"files"`
}

// Add the containers log file to the files of this config
func (config *LogstashForwarderConfig) AddContainerLogFile(container *docker.Container) {
	id := container.ID
	file := File{}
	file.Paths = []string{fmt.Sprintf("/var/lib/docker/containers/%s/%s-json.log", id, id)}
	file.Fields = make(map[string]string)
	file.Fields["type"] = "docker"
	file.Fields["docker.id"] = id
	file.Fields["docker.hostname"] = container.Config.Hostname
	file.Fields["docker.name"] = container.Name
	file.Fields["docker.image"] = container.Config.Image

	config.Files = append(config.Files, file)
}

// Create a new logstash-forwarder config from a file
func NewFromFile(path string) (*LogstashForwarderConfig, error) {
	configFile, err := os.Open(path)
	defer configFile.Close()
	if err != nil {
		return nil, err
	}

	logstashConfig := new(LogstashForwarderConfig)

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&logstashConfig); err != nil {
		return nil, err
	}

	return logstashConfig, nil
}

// Create a new logstash-forwarder default config
func NewFromDefault(logstashEndpoint string) *LogstashForwarderConfig {
	network := Network{
		Servers:        []string{logstashEndpoint},
		SslCertificate: "/mnt/logstash-forwarder/logstash-forwarder.crt",
		SslKey:         "/mnt/logstash-forwarder/logstash-forwarder.key",
		SslCa:          "/mnt/logstash-forwarder/logstash-forwarder.crt",
		Timeout:        15,
	}

	config := &LogstashForwarderConfig{
		Network: network,
		Files:   []File{},
	}

	return config
}

// Create a new logstash-forwarder config for a container if it contains a config file
// at /etc/logstash-forwarder.conf
func NewFromContainer(container *docker.Container) (*LogstashForwarderConfig, error) {
	containerDirectory := utils.ContainerFileSystemPath(container)
	path := fmt.Sprintf("%s/etc/logstash-forwarder.conf", containerDirectory)
	config, err := NewFromFile(path)
	if err != nil {
		return nil, err
	}
	log.Printf("Found logstash-forwarder config in %s", container.ID)

	for _, file := range config.Files {
		log.Printf("Adding files %s of type %s", file.Paths, file.Fields["type"])
		for i, path := range file.Paths {
			file.Paths[i] = containerDirectory + path
		}
	}
	return config, nil
}
