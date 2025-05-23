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
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	// htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudbridgeuy/puper/pkg/display"
	"github.com/cloudbridgeuy/puper/pkg/errors"
	"github.com/cloudbridgeuy/puper/pkg/geckodriver"
	"github.com/cloudbridgeuy/puper/pkg/html"
	"github.com/cloudbridgeuy/puper/pkg/logger"
	"github.com/cloudbridgeuy/puper/pkg/net"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "puper [STDIN/FILE/URL]",
	Short: "Manipulate HTML read from a file, stdin, or an url",
	Long: `
Puper
-----

Tool that allows you to filter or select portions of your HTML code,
which you can then clean up by removing all attributes or annoying span
fields. This is quite useful for pseudo-RAG enabled AI scripts, where you
don't want to send a bunch of garbage to the LLMs.

Puper uses 'firefox' and 'geckodriver' to get the source of the provided
URL, allowing you to render pages that require client-side JavaScript
rendering. Each call will spawn a new instance of both resources, listening
on a random open port of your machine (by default), so you can run multiple
instances of 'puper' at the same time without issues (other than your
hardware's resources).`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the verbose flag")
			return
		}

		if verbose {
			logger.Verbose()
		}

		var inputReader io.Reader = cmd.InOrStdin()

		if len(args) == 0 {
			args = []string{"-"}
		}

		selectors, err := cmd.Flags().GetStringSlice("selector")
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the selector flag")
			return
		}

		wait, err := cmd.Flags().GetInt("wait")
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the wait flag")
			return
		}

		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			if err != nil {
				errors.HandleAsPuperError(err, "Can't get the port flag")
				return
			}
		}

		firefoxBinary, err := cmd.Flags().GetString("firefox-binary")
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the firefox-binary flag")
			return
		}

		if port == 0 {
			port, err = net.GetRandomUnusedPort()
			if err != nil {
				errors.HandleAsPuperError(err, "Can't get a random unused port from the OS")
				return
			}
		}

		markdown, err := cmd.Flags().GetBool("markdown")
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the markdown flag")
			return
		}

		remove, err := cmd.Flags().GetString("remove")
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the remove flag")
			return
		}

		// Check if the entrypoint is a URL
		if strings.HasPrefix(args[0], "http://") || strings.HasPrefix(args[0], "https://") {
			logger.Logger.Debugf("Running geckodriver")
			g := geckodriver.NewGeckodriverBuilder().
				WithUrl(args[0]).
				WithSelectors(selectors).
				WithPort(port).
				WithBinary(firefoxBinary).
				WithDefaultLogger().
				WithWait(wait).
				Build()

			err = g.Run()
			if err != nil {
				errors.HandleAsPuperError(err, "Geckodriver failed to fetch the page source")
				return
			}

			inputReader = strings.NewReader(g.GetSource())
		} else if args[0] != "-" {
			file, err := os.Open(args[0])
			if err != nil {
				errors.HandleAsPuperError(err, "Can't open file")
				return
			}
			inputReader = file
		}

		charset, err := cmd.Flags().GetString("charset")
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the charset flag")
			return
		}

		root, err := html.ParseHTML(inputReader, charset)
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the html document")
			return
		}

		selectedNodes, err := html.Get(root, selectors)
		if err != nil {
			errors.HandleAsPuperError(err, "Can't run selectors on root")
			return
		}

		removeAttributes, err := cmd.Flags().GetBool("remove-attributes")
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the remove-attributes flag")
			return
		}
		removeSpan, err := cmd.Flags().GetBool("remove-span")
		if err != nil {
			errors.HandleAsPuperError(err, "Can't get the remove-span flag")
			return
		}

		b := display.NewDisplayBuilder().
			WithAttributes(!removeAttributes).
			WithSpan(!removeSpan)

		if markdown {
			var buffer bytes.Buffer
			b.WithWriter(&buffer).Build().Print(selectedNodes)

			conv := converter.NewConverter(
				converter.WithPlugins(
					base.NewBasePlugin(),
					commonmark.NewCommonmarkPlugin(
						commonmark.WithStrongDelimiter("**"),
					),
					strikethrough.NewStrikethroughPlugin(),
					table.NewTablePlugin(),
				),
			)

			conv.Register.TagType("button", converter.TagTypeRemove, converter.PriorityStandard)

			for _, r := range strings.Split(remove, ",") {
				buffer = *bytes.NewBuffer([]byte(strings.Replace(buffer.String(), r, "", -1)))
			}

			m, err := conv.ConvertReader(&buffer)
			if err != nil {
				errors.HandleAsPuperError(err, "Can't convert the HTML to markdown")
				return
			}

			fmt.Println(string(m))
		} else {
			b.Build().Print(selectedNodes)
		}

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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.puper.yaml)")

	rootCmd.Flags().StringP("charset", "c", "", "Charset")
	rootCmd.Flags().String("firefox-binary", "/Applications/Firefox.app/Contents/MacOS/firefox", "Firefox binary path")
	rootCmd.Flags().Int("wait", 1, "Time to wait for a page to render if an URL was provided")
	rootCmd.Flags().Int("port", 0, "Geckodriver port. A random one will be selected if empty.")
	rootCmd.Flags().StringSliceP("selector", "s", []string{"*"}, "CSS Selector")
	rootCmd.Flags().Bool("remove-attributes", false, "Remove attributes")
	rootCmd.Flags().Bool("remove-span", false, "Remove span")
	rootCmd.Flags().Bool("verbose", false, "Verbose output")
	rootCmd.Flags().Bool("markdown", false, "Convert the output to markdown")
	rootCmd.Flags().String("remove", "<<", "Comma separated list of strings to remove. Useful for markdown parsing.")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".puper")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
