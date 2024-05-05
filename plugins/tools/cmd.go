package tools

import (
	"fmt"
	"runtime"

	"github.com/Pishia-IA/core/config"
	log "github.com/sirupsen/logrus"
)

type Cmd struct {
}

func NewCmd(config *config.Base) *Cmd {
	return &Cmd{}
}

func (c *Cmd) Run(params map[string]interface{}) (string, error) {
	log.Debugf("Running the CMD tool with the following parameters: %v", params)
	return "", nil
}

func (c *Cmd) Setup() error {
	return nil
}

func (c *Cmd) NeedConfirmation() bool {
	return true
}

func (c *Cmd) Description() string {
	os := runtime.GOOS
	switch os {
	case "darwin":
		os = "macOS"
	case "linux":
		os = "Linux"
	case "windows":
		os = "Windows"
	}
	return fmt.Sprintf("CMD is a tool that allows you to run commands in the terminal. I will use commands compatible with %s", os)
}

func (c *Cmd) Parameters() map[string]*ToolParameter {
	return map[string]*ToolParameter{
		"command": {
			Type:        "string",
			Format:      "",
			Required:    true,
			Description: "The command that you want to run in the terminal.",
		},
	}
}

func (c *Cmd) UseCase() []string {
	return []string{
		"Run this tool if the user wants to run a command in the terminal.",
	}
}
