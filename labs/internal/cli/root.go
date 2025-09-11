package cli

import (
	"d7024e/pkg/build"
	"fmt"
	"net"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const TimeLayout = "2006-01-02 15:04:05"

var Verbose bool

var localIP = ""

func getLocalIP() (string, error) {
	// 1) Try DNS for container hostname (matches Docker DNS)
	if hn, err := os.Hostname(); err == nil {
		if ips, err := net.LookupHost(hn); err == nil && len(ips) > 0 {
			return ips[0], nil
		}
	}
	return "", nil
}

type DefaultFieldHook struct{}

func (hook *DefaultFieldHook) Fire(entry *log.Entry) error {

	entry.Data["containerId"] = localIP
	return nil
}

// Levels returns the log levels for which this hook is triggered.
func (hook *DefaultFieldHook) Levels() []log.Level {
	return log.AllLevels
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
}

var rootCmd = &cobra.Command{
	Use:   "kadlab",
	Short: "kadlab",
	Long:  "kadlab",
}

func Execute() {
	_localIP, err := getLocalIP()
	if err != nil {
		log.Fatalf("Failed to get local IP: %v", err)
	}
	localIP = _localIP
	// Set log level
	log.SetLevel(log.DebugLevel)
	log.AddHook(&DefaultFieldHook{})

	// Log buildtime and version
	log.Infof("Build version: %s", build.BuildVersion)
	log.Infof("Build time: %s", build.BuildTime)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
