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
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tebeka/selenium"
	"golang.org/x/net/html"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "puper [STDIN/FILE/URL]",
	Short: "Clean up HTML code",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		verbose, err := cmd.Flags().GetBool("verbose")
		handleError(err)

		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		var inputReader io.Reader = cmd.InOrStdin()

		if len(args) == 0 {
			args = []string{"-"}
		}

		selectors, err := cmd.Flags().GetStringSlice("selector")
		handleError(err)

		wait, err := cmd.Flags().GetInt("wait")
		handleError(err)

		port, err := cmd.Flags().GetInt("port")
		handleError(err)

		if port == 0 {
			port, err = getRandomUnusedPort()
			handleError(err)
		}

		// Check if the entrypoint is a URL
		if strings.HasPrefix(args[0], "http://") || strings.HasPrefix(args[0], "https://") {
			log.Debug("Prepare the geckodriver command.")
			command := exec.Command("geckodriver")
			command.Env = append(os.Environ(), "MOZ_HEADLESS=1", "MOZ_REMOTE_SETTINGS_DEVTOOLS=1")
			command.Args = append(command.Args, fmt.Sprintf("--port=%d", port), "-b", "/Applications/Firefox.app/Contents/MacOS/firefox")

			log.Debug(fmt.Sprintf("Running command: %s", strings.Join(command.Args, " ")))

			log.Debug(fmt.Sprintf("Launch geckodriver on port %d", port))
			if err := command.Start(); err != nil {
				log.Error("Failed to start geckodriver: %v", err)
			}
			defer func() {
				log.Debug("Killing geckodriver")
				command.Process.Kill()
			}()

			log.Debug("Checking for Firefox process...")
			timeoutDuration := 10 * time.Second
			sleepInterval := 500 * time.Millisecond
			startTime := time.Now()

			for {
				if time.Since(startTime) >= timeoutDuration {
					log.Error("Timeout: Failed to detect a running Firefox instance.")
					return
				}

				processes, err := process.Processes()
				if err != nil {
					log.Error(fmt.Sprintf("Failed to get processes: %v", err))
				}

				for _, p := range processes {
					name, err := p.Name()
					if err == nil && name == "firefox" {
						log.Debug("Headless Firefox instance detected.")
						goto WebDriverSetup
					}
				}

				time.Sleep(sleepInterval)
			}
		WebDriverSetup:
			url := fmt.Sprintf("http://localhost:%d", port)
			caps := selenium.Capabilities{"browserName": "firefox"}
			log.Debug(fmt.Sprintf("Create webdriver client connection to: %s", url))
			wd, err := selenium.NewRemote(caps, url)
			defer func() {
				log.Debug("Quitting webdriver client")
				wd.Quit()
			}()
			if err != nil {
				log.Error(fmt.Sprintf("Failed to create WebDriver client: %v", err))
				return
			}

			log.Debug("Getting webpage")
			err = wd.Get(args[0])
			if err != nil {
				log.Error("Failed to load URL: %v", err)
			}

			if len(selectors) > 0 && selectors[0] != "*" && selectors[0] != "" {
				log.Debug(fmt.Sprintf("Wait for locator: %s", selectors[0]))
				_, err := wd.FindElement(selenium.ByCSSSelector, selectors[0])
				if err != nil {
					log.Error("Failed to find element: %v", err)
				}
			} else {
				log.Debug("Wait for %d seconds", wait)
				time.Sleep(time.Duration(wait) * time.Second)
			}

			source, err := wd.PageSource()
			if err != nil {
				log.Error("Failed to get page source: %v", err)
			}

			inputReader = strings.NewReader(source)
		} else if args[0] != "-" {
			file, err := os.Open(args[0])
			handleError(err)
			inputReader = file
		}

		charset, err := cmd.Flags().GetString("charset")
		handleError(err)

		root, err := ParseHTML(inputReader, charset)
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
					log.Error(fmt.Sprintf("Selector parsing error: %s\n", err.Error()))
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
	rootCmd.Flags().Int("wait", 1, "Time to wait for a page to render if an URL was provided")
	rootCmd.Flags().Int("port", 0, "Geckodriver port. A random one will be selected if empty.")
	rootCmd.Flags().StringSliceP("selector", "s", []string{"*"}, "CSS Selector")
	rootCmd.Flags().Bool("remove-attributes", false, "Remove attributes")
	rootCmd.Flags().Bool("remove-span", false, "Remove span")
	rootCmd.Flags().Bool("verbose", false, "Verbose output")
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

func getRandomUnusedPort() (int, error) {
	// Create a listener on port 0
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	// Retrieve the assigned port
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
