package config

import (
	"flag"
	"strings"
)

type Measures []string

func (m *Measures) String() string {
	return "[" + strings.Join(*m, ", ") + "]"
}

func (i *Measures) Set(v string) error {
	if v == "all" {
		*i = []string{"all"}
		return nil
	}

	for _, it := range *i {
		// if has all option don't add anything else
		if it == "all" {
			*i = []string{"all"}
			return nil
		}
		if it == v {
			// don't add dups
			return nil
		}
	}

	if strings.Contains(v, ",") {
		if strings.Contains(v, "all") {
			*i = []string{"all"}
			return nil
		}
		*i = append(*i, strings.Split(v, ",")...)
		return nil
	}

	*i = append(*i, v)
	return nil
}

type Types []string

func (t *Types) String() string {
	return "[" + strings.Join(*t, ", ") + "]"
}

func (t *Types) Set(v string) error {
	if strings.Contains(v, ",") {
		*t = append(*t, strings.Split(v, ",")...)
		return nil
	}

	*t = append(*t, v)
	return nil
}

type Config struct {
	FormatImports *bool
	ShowVersion   *bool
	Types         *Types
	Measures      *Measures
	FilePath      *string

	flagSet *flag.FlagSet
}

func NewConfig(name string) *Config {
	cfg := &Config{
		FormatImports: new(bool),
		ShowVersion:   new(bool),
		Types:         new(Types),
		Measures:      new(Measures),
		FilePath:      new(string),
		// TODO: does this need to be more configurable?
		flagSet: flag.NewFlagSet(name, flag.ExitOnError),
	}

	// should accept multipe targets
	cfg.flagSet.Var(cfg.Types, "t", `List of target interface(s)/struct(s). 
Can be repeated like '-t MyInterface -t MyStruct'
Can also be comma seprated like '-t MyInterface,MyStruct'`)

	cfg.flagSet.Var(cfg.Measures, "m", `Measures to include. 
Possible values [all, duration, total, success, error].
Can be reapeted like '-m duration -m total'
Can also be comma seprated '-m duration,total'
If 'all' is specified others will be ignored`)

	cfg.flagSet.BoolVar(cfg.FormatImports, "fmt", true, "If set to true, will run imports.Process on the generated wrapper")
	cfg.flagSet.BoolVar(cfg.ShowVersion, "version", false, "Show program version")
	cfg.flagSet.BoolVar(cfg.ShowVersion, "v", false, "Show program version")
	cfg.flagSet.StringVar(cfg.FilePath, "f", "", "File path to parse. Can be overwritten with GOFILE")

	return cfg
}

func (c *Config) Parse(args []string) error {
	return c.flagSet.Parse(args)
}
