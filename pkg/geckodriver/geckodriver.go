package geckodriver

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/cloudbridgeuy/puper/pkg/errors"
	"github.com/cloudbridgeuy/puper/pkg/logger"
	"github.com/shirou/gopsutil/process"
	"github.com/tebeka/selenium"
)

type Geckodriver struct {
	binary    string
	port      int
	logger    *log.Logger
	url       string
	selectors []string
	wait      int
	source    string
}

// geckodriverConfigFn is a function that configures a Geckodriver.
type geckodriverConfigFn func(g Geckodriver) Geckodriver

// NewGeckodriver creates a new Geckodriver with the given configuration.
func NewGeckodriver(fns ...geckodriverConfigFn) Geckodriver {
	var g Geckodriver

	for _, f := range fns {
		g = f(g)
	}

	return g
}

// WithDefaultLogger sets the default logger instance on the Geckodriver struct.
func WithDefaultLogger() geckodriverConfigFn {
	return func(g Geckodriver) Geckodriver {
		g.logger = logger.Logger
		return g
	}
}

// WithBinary sets the binary for the Geckodriver.
func WithBinary(binary string) geckodriverConfigFn {
	return func(g Geckodriver) Geckodriver {
		g.binary = binary
		return g
	}
}

// WithPort sets the port for the Geckodriver.
func WithPort(port int) geckodriverConfigFn {
	return func(g Geckodriver) Geckodriver {
		g.port = port
		return g
	}
}

// WithSelectors sets the selectors for the Geckodriver.
func WithSelectors(selectors []string) geckodriverConfigFn {
	return func(g Geckodriver) Geckodriver {
		g.selectors = selectors
		return g
	}
}

// WithUrl sets the URL for the Geckodriver.
func WithUrl(url string) geckodriverConfigFn {
	return func(g Geckodriver) Geckodriver {
		g.url = url
		return g
	}
}

// WithWait sets the URL for the Geckodriver.
func WithWait(wait int) geckodriverConfigFn {
	return func(g Geckodriver) Geckodriver {
		g.wait = wait
		return g
	}
}

func (g *Geckodriver) Run() error {
	g.logger.Debug("Prepare the geckodriver command.")
	command := exec.Command("geckodriver")
	command.Env = append(os.Environ(), "MOZ_HEADLESS=1", "MOZ_REMOTE_SETTINGS_DEVTOOLS=1")
	command.Args = append(command.Args, fmt.Sprintf("--port=%d", g.port), "-b", g.binary)

	g.logger.Debug("", "$", strings.Join(command.Args, " "))
	if err := command.Start(); err != nil {
		return errors.NewPuperError(err, "Failed to start geckodriver")
	}

	defer func() {
		g.logger.Debug("Killing geckodriver")
		command.Process.Kill()
	}()

	g.logger.Debug("Checking for Firefox process")
	timeoutDuration := 10 * time.Second
	sleepInterval := 500 * time.Millisecond
	startTime := time.Now()

	for {
		if time.Since(startTime) >= timeoutDuration {
			return errors.NewPuperError(fmt.Errorf("Timeout"), "Failed to detect a running Firefox instance")
		}

		processes, err := process.Processes()
		if err != nil {
			return errors.NewPuperError(err, "Failed to get processes")
		}

		for _, p := range processes {
			name, err := p.Name()
			if err == nil && name == "firefox" {
				g.logger.Debug("Headless Firefox instance detected")
				return g.webdriver()
			}
		}

		time.Sleep(sleepInterval)
	}
}

func (g *Geckodriver) webdriver() error {
	g.logger.Debug("Starting firefox control through geckodriver using the webdriver protocol")

	url := fmt.Sprintf("http://localhost:%d", g.port)
	caps := selenium.Capabilities{"browserName": "firefox"}

	g.logger.Debug("Creating webdriver client connection", "url", url)
	wd, err := selenium.NewRemote(caps, url)
	defer func() {
		g.logger.Debug("Quitting webdriver client")
		wd.Quit()
	}()

	if err != nil {
		return errors.NewPuperError(err, "Failed to create WebDriver client")
	}

	g.logger.Debug("Getting webpage")
	err = wd.Get(g.url)
	if err != nil {
		return errors.NewPuperError(err, "Failed to load URL")
	}

	if len(g.selectors) > 0 && g.selectors[0] != "*" && g.selectors[0] != "" {
		g.logger.Debug("Waiting for locator", "selector", g.selectors[0])
		_, err := wd.FindElement(selenium.ByCSSSelector, g.selectors[0])
		if err != nil {
			return errors.NewPuperError(err, "Failed to find element")
		}
	} else {
		g.logger.Debug("Waiting for page to load", "seconds", g.wait)
		time.Sleep(time.Duration(g.wait) * time.Second)
	}

	g.source, err = wd.PageSource()
	if err != nil {
		return errors.NewPuperError(err, "Failed to get page source")
	}

	return nil
}

// GetSource returns the source found after running the `Run` method.
func (g Geckodriver) GetSource() string {
	return g.source
}
