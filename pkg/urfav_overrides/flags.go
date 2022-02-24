package cfflags

import (
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

type Flags struct {
	*flag.FlagSet
	urFavFlags []cli.Flag
}

func New(name string, flags []cli.Flag, c *cli.Context) (*Flags, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	for _, f := range flags {
		if err := f.Apply(set); err != nil {
			return nil, err
		}
	}
	set.SetOutput(ioutil.Discard)
	ca := []string{}
	ca = append(ca, c.Args().Slice()...)
	// context.Args() for this command will ONLY contain the role and any flags provided after the role
	// this slice of os.Args will only contain flags and not the role if it was provided
	ag := []string{}
	ag = append(ag, os.Args[1:len(os.Args)-len(ca)]...)
	ag = append(ag, ca[1:]...)
	err := normalizeFlags(flags, set)
	if err != nil {
		return nil, err
	}
	err = set.Parse(ag)
	wow := os.Args
	_ = wow
	if err != nil {
		return nil, err
	}
	return &Flags{FlagSet: set, urFavFlags: flags}, nil
}
func copyFlag(name string, ff *flag.Flag, set *flag.FlagSet) {
	switch ff.Value.(type) {
	case cli.Serializer:
		_ = set.Set(name, ff.Value.(cli.Serializer).Serialize())
	default:
		_ = set.Set(name, ff.Value.String())
	}
}

func normalizeFlags(flags []cli.Flag, set *flag.FlagSet) error {
	visited := make(map[string]bool)
	set.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	for _, f := range flags {
		parts := f.Names()
		if len(parts) == 1 {
			continue
		}
		var ff *flag.Flag
		for _, name := range parts {
			name = strings.Trim(name, " ")
			if visited[name] {
				if ff != nil {
					return errors.New("Cannot use two forms of the same flag: " + name + " " + ff.Name)
				}
				ff = set.Lookup(name)
			}
		}
		if ff == nil {
			continue
		}
		for _, name := range parts {
			name = strings.Trim(name, " ")
			if !visited[name] {
				copyFlag(name, ff, set)
			}
		}
	}
	return nil
}
func (set *Flags) searchFS(name string) []string {
	for _, f := range set.urFavFlags {
		for _, n := range f.Names() {
			if n == name {
				return f.Names()
			}
		}
	}
	return nil
}
func (set *Flags) String(name string) string {
	names := set.searchFS(name)
	for _, n := range names {
		f := set.Lookup(n)
		if f != nil {
			parsed := f.Value.String()
			if parsed != "" {
				return parsed
			}
		}
	}
	return ""
}

func (set *Flags) Bool(name string) bool {
	names := set.searchFS(name)
	for _, n := range names {
		f := set.Lookup(n)
		if f != nil {
			parsed, _ := strconv.ParseBool(f.Value.String())
			if parsed {
				return parsed
			}
		}
	}
	return false
}
