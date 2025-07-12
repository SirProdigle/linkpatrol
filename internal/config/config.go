package config

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Dir               string
	Watch             bool
	Concurrency       int
	TesterConcurrency int
	Timeout           time.Duration
	Rate              int
	ConfigFile        string
	Verbose           bool
	TermWidth         int
	NoTruncate        bool
	CPUProfile        string
	MemProfile        string
	Target            string
}

func NewConfig() Config {
	return Config{}
}

func (c *Config) InitFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()
	f.StringP("dir", "d", ".", "root directory to scan")
	f.BoolP("watch", "w", false, "enable live watch mode")
	f.StringP("config", "c", "", "path to config file")
	f.IntP("concurrency", "n", runtime.NumCPU()*2, "max concurrent file readers")
	f.IntP("tester-concurrency", "", 100, "max concurrent link testers (HTTP requests)")
	f.DurationP("timeout", "t", 5*time.Second, "per-request timeout")
	f.IntP("rate", "r", 10, "max requests per second per domain")
	f.BoolP("verbose", "v", false, "enable verbose logging")
	f.IntP("width", "", 0, "terminal width override (0 = auto-detect)")
	f.BoolP("no-truncate", "", false, "don't truncate URLs or error messages")
	f.StringP("cpuprofile", "", "", "write cpu profile to file")
	f.StringP("memprofile", "", "", "write memory profile to file")
	f.StringP("target", "x", "", "target URL to scan")
	viper.BindPFlag("dir", f.Lookup("dir"))
	viper.BindPFlag("watch", f.Lookup("watch"))
	viper.BindPFlag("config", f.Lookup("config"))
	viper.BindPFlag("concurrency", f.Lookup("concurrency"))
	viper.BindPFlag("tester-concurrency", f.Lookup("tester-concurrency"))
	viper.BindPFlag("timeout", f.Lookup("timeout"))
	viper.BindPFlag("rate", f.Lookup("rate"))
	viper.BindPFlag("verbose", f.Lookup("verbose"))
	viper.BindPFlag("width", f.Lookup("width"))
	viper.BindPFlag("no-truncate", f.Lookup("no-truncate"))
	viper.BindPFlag("cpuprofile", f.Lookup("cpuprofile"))
	viper.BindPFlag("memprofile", f.Lookup("memprofile"))
	viper.BindPFlag("target", f.Lookup("target"))
	viper.SetEnvPrefix("linkpatrol")
	viper.AutomaticEnv()

	if cfg := viper.GetString("config"); cfg != "" {
		viper.SetConfigFile(cfg)
	} else {
		viper.SetConfigName("linkpatrol")
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		}
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}

func (c *Config) LoadFromViper() {
	c.Dir = viper.GetString("dir")
	c.Watch = viper.GetBool("watch")
	c.Concurrency = viper.GetInt("concurrency")
	c.TesterConcurrency = viper.GetInt("tester-concurrency")
	c.Timeout = viper.GetDuration("timeout")
	c.Rate = viper.GetInt("rate")
	c.ConfigFile = viper.GetString("config")
	c.Verbose = viper.GetBool("verbose")
	c.TermWidth = viper.GetInt("width")
	c.NoTruncate = viper.GetBool("no-truncate")
	c.CPUProfile = viper.GetString("cpuprofile")
	c.MemProfile = viper.GetString("memprofile")
	c.Target = viper.GetString("target")
}

// Print is deprecated. Use logger.Config() instead for consistent logging.
func (c *Config) Print() {
	fmt.Printf("Starting LinkPatrol with:\n")
	fmt.Printf("  Directory: %s\n", c.Dir)
	fmt.Printf("  Watch mode: %t\n", c.Watch)
	fmt.Printf("  Walker concurrency: %d\n", c.Concurrency)
	fmt.Printf("  Tester concurrency: %d\n", c.TesterConcurrency)
	fmt.Printf("  Timeout: %v\n", c.Timeout)
	fmt.Printf("  Rate limit: %d req/s\n", c.Rate)
	fmt.Printf("  Cache enabled: %t\n", true)
}
