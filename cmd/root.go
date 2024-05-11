/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/Pishia-IA/core/core"
	"github.com/Pishia-IA/core/plugins/assistants"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

type newline struct{ tok string }

func (n *newline) Scan(state fmt.ScanState, verb rune) error {
	tok, err := state.Token(false, func(r rune) bool {
		return r != '\n'
	})
	if err != nil {
		return err
	}
	if _, _, err := state.ReadRune(); err != nil {
		if len(tok) == 0 {
			panic(err)
		}
	}
	n.tok = string(tok)
	return nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pishia",
	Short: "Pishia is CLI for creating custom personal assistant for your own use.",
	Long: `Pishia is an open source alternative to Google Assistant, Amazon Alexa, and Apple Siri.
It is a CLI tool that allows you to create your own personal assistant with custom commands and responses.
You can use it to automate tasks, get information, and more.`,
}

// cli is an action that you can use to run the CLI.
var cliCmd = &cobra.Command{
	Use:   "cli",
	Short: "Run the Pishia CLI",
	Long:  `Run the Pishia CLI to create your own personal assistant with custom commands and responses.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetLevel(log.DebugLevel)
		err := core.Boot()
		if err != nil {
			cmd.Println("Error booting the core:", err)
			return
		}
		assistant := assistants.GetDefaultAssistant()
		err = assistant.Setup()

		if err != nil {
			cmd.Println("Error setting up the Ollama:", err)
			return
		}

		for {
			cmd.Print("You: ")
			var n newline
			fmt.Scan(&n)

			cmd.Print("Pishia: ")

			if err != nil {
				cmd.Println("Error reading input:", err)
				return
			}

			printResponse := func(output string, err error) {
				if err != nil {
					cmd.Println("Error sending request:", err)
					return
				}

				cmd.Print(output)
			}

			err := assistant.SendRequest(n.tok, printResponse)

			if err != nil {
				cmd.Println("Error sending request:", err)
				return
			}

			cmd.Println()
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Add the CLI command.
	rootCmd.AddCommand(cliCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}

}
