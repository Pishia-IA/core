package core

import (
	"github.com/Pishia-IA/core/config"
	"github.com/Pishia-IA/core/plugins/assistants"
	"github.com/Pishia-IA/core/plugins/tools"
)

// Boot starts the core.
func Boot() error {
	config, err := config.GetCurrentConfigurations()
	if err != nil {
		return err
	}

	tools.StartTools(config)
	assistants.StartAssistants(config)
	return nil
}
