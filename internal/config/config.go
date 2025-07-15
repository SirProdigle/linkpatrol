package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Dir         string
	Watch       bool
	Concurrency int
	Timeout     time.Duration
	Rate        int
	ConfigFile  string
	Verbose     bool
	TermWidth   int
	NoTruncate  bool
	CPUProfile  string
	MemProfile  string
	Target      string
}

func NewConfig() Config {
	return Config{}
}

func (c *Config) InitFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()
	f.StringP("config", "c", "", "path to config file")
	f.IntP("concurrency", "n", 50, "max concurrent web crawlers & testers")
	f.DurationP("timeout", "", 30*time.Second, "per-request timeout")
	f.IntP("rate", "r", 20, "max requests per second per domain")
	f.BoolP("verbose", "v", false, "enable verbose logging")
	f.IntP("width", "", 0, "terminal width override (0 = auto-detect)")
	f.BoolP("no-truncate", "", false, "don't truncate URLs or error messages")
	f.StringP("cpuprofile", "", "", "write cpu profile to file")
	f.StringP("memprofile", "", "", "write memory profile to file")

	// Remove directory and watch flags as they're not needed for web crawling
	f.StringP("dir", "d", ".", "root directory to scan")
	f.BoolP("watch", "w", false, "enable live watch mode")
	viper.BindPFlag("target", f.Lookup("target"))
	viper.BindPFlag("config", f.Lookup("config"))
	viper.BindPFlag("concurrency", f.Lookup("concurrency"))
	viper.BindPFlag("timeout", f.Lookup("timeout"))
	viper.BindPFlag("rate", f.Lookup("rate"))
	viper.BindPFlag("verbose", f.Lookup("verbose"))
	viper.BindPFlag("width", f.Lookup("width"))
	viper.BindPFlag("no-truncate", f.Lookup("no-truncate"))
	viper.BindPFlag("cpuprofile", f.Lookup("cpuprofile"))
	viper.BindPFlag("memprofile", f.Lookup("memprofile"))

	// Keep these for backward compatibility but deprecate them
	viper.BindPFlag("dir", f.Lookup("dir"))
	viper.BindPFlag("watch", f.Lookup("watch"))
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
	c.Concurrency = viper.GetInt("concurrency")
	c.Timeout = viper.GetDuration("timeout")
	c.Rate = viper.GetInt("rate")
	c.ConfigFile = viper.GetString("config")
	c.Verbose = viper.GetBool("verbose")
	c.TermWidth = viper.GetInt("width")
	c.NoTruncate = viper.GetBool("no-truncate")
	c.CPUProfile = viper.GetString("cpuprofile")
	c.MemProfile = viper.GetString("memprofile")
}
