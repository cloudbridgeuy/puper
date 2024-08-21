/*
Copyright © 2024 Guzmán Monné guzman.monne@cloudbridge.com.uy

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/html"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "puper [HTML]",
	Short: "Clean up HTML code",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var inputReader io.Reader = cmd.InOrStdin()

		if len(args) > 0 && args[0] != "-" {
			file, err := os.Open(args[0])
			handleError(err)
			inputReader = file
		}

		charset, err := cmd.Flags().GetString("charset")
		handleError(err)

		root, err := ParseHTML(inputReader, charset)
		handleError(err)

		selectors, err := cmd.Flags().GetStringSlice("selector")
		handleError(err)

		// Parse the selectors
		selectorFuncs := []SelectorFunc{}
		funcGenerator := Select
		var selector string
		for len(selectors) > 0 {
			selector, selectors = selectors[0], selectors[1:]
			switch selector {
			case "*": // select all
				continue
			case ">":
				funcGenerator = SelectFromChildren
			case "+":
				funcGenerator = SelectNextSibling
			case ",": // nil will signify a comma
				selectorFuncs = append(selectorFuncs, nil)
			default:
				selector, err := ParseSelector(selector)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Selector parsing error: %s\n", err.Error())
					os.Exit(2)
				}
				selectorFuncs = append(selectorFuncs, funcGenerator(selector))
				funcGenerator = Select
			}
		}

		selectedNodes := []*html.Node{}
		currNodes := []*html.Node{root}
		for _, selectorFunc := range selectorFuncs {
			if selectorFunc == nil { // hit a comma
				selectedNodes = append(selectedNodes, currNodes...)
				currNodes = []*html.Node{root}
			} else {
				currNodes = selectorFunc(currNodes)
			}
		}
		selectedNodes = append(selectedNodes, currNodes...)

		removeAttributes, err := cmd.Flags().GetBool("remove-attributes")
		handleError(err)
		removeSpan, err := cmd.Flags().GetBool("remove-span")
		handleError(err)

		Display{
			attributes: !removeAttributes,
			span:       !removeSpan,
		}.Print(selectedNodes)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.puper.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("charset", "c", "", "Charset")
	rootCmd.Flags().StringSliceP("selector", "s", []string{"*"}, "CSS Selector")
	rootCmd.Flags().Bool("remove-attributes", false, "Remove attributes")
	rootCmd.Flags().Bool("remove-span", false, "Remove span")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".puper" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".puper")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
