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

type geckodriver struct {
	binary    string
	port      int
	logger    *log.Logger
	url       string
	selectors []string
	wait      int
	source    string
}

type builder struct {
	inner *geckodriver
}

func NewGeckodriverBuilder() *builder {
	return &builder{
		inner: &geckodriver{},
	}
}

// WithDefaultLogger sets the default logger instance on the Geckodriver struct.
func (b *builder) WithDefaultLogger() *builder {
	b.inner.logger = logger.Logger
	return b
}

// WithLogger sets the default logger instance on the Geckodriver struct.
func (b *builder) WithLogger(log.Logger) *builder {
	b.inner.logger = logger.Logger
	return b
}

// WithBinary sets the binary for the Geckodriver.
func (b *builder) WithBinary(binary string) *builder {
	b.inner.binary = binary
	return b
}

// WithPort sets the port for the Geckodriver.
func (b *builder) WithPort(port int) *builder {
	b.inner.port = port
	return b
}

// WithSelectors sets the selectors for the Geckodriver.
func (b *builder) WithSelectors(selectors []string) *builder {
	b.inner.selectors = selectors
	return b
}

// WithUrl sets the URL for the Geckodriver.
func (b *builder) WithUrl(url string) *builder {
	b.inner.url = url
	return b
}

// WithWait sets the URL for the Geckodriver.
func (b *builder) WithWait(wait int) *builder {
	b.inner.wait = wait
	return b
}

// Build returns the inner struct
func (b *builder) Build() *geckodriver {
	return b.inner
}

func (g *geckodriver) Run() error {
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

func (g *geckodriver) webdriver() error {
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
		wd.SetImplicitWaitTimeout(time.Duration(g.wait) * time.Second)
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
func (g geckodriver) GetSource() string {
	return g.source
}
