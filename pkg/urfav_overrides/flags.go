package cfflags

import (
	"errors"
	"flag"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

type Flags struct {
	FlagSet    *flag.FlagSet
	urFavFlags []cli.Flag
}

// The purpose of this package is to allow the assume cli command to accept flags on either side of the "role" arg
// for example, `assume -c my-role -region=us-east-1` by default, urfav-cli, the cli framework that we are using does not
// support this usage pattern.
//
// We have extracted some methods from the original urfav-cli library to mimic the original behaviour but processing all the flags.
// to use this in a command,
// This package interacts with os.Args directly
//
// allFlags := cfflags.New("name",GlobalFlagsList, c, 1)
// allFlags.String("region")
// When using granted global flags for a subcommand. Eg. granted registry add {URL:} -c ref. you will need to need to specify a command depth
// for it to work correctly.
// The depth is used to filter out the subcommands when grabbing out the flags passed in the body of the cli request
// err = set.Parse(ag) fails to populate the flag value when args includes subcommands so we filter them out
func New(name string, flags []cli.Flag, c *cli.Context, depth int) (*Flags, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	for _, f := range flags {
		if err := f.Apply(set); err != nil {
			return nil, err
		}
	}

	set.SetOutput(io.Discard)

	ca := []string{}
	if c.Args().Len() > 1 {
		// append the flags excluding the role arg
		ca = append(ca, c.Args().Slice()[1:]...)
	}

	// context.Args() for this command will ONLY contain the role and any flags provided after the role
	// this slice of os.Args will only contain flags and not the role if it was provided
	ag := []string{}
	ag = append(ag, os.Args[depth:len(os.Args)-c.Args().Len()]...)
	ag = append(ag, ca...)

	err := normalizeFlags(flags, set)
	if err != nil {
		return nil, err
	}
	err = set.Parse(ag)
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
		f := set.FlagSet.Lookup(n)
		if f != nil {
			parsed := f.Value.String()
			if parsed != "" {
				return parsed
			}
		}
	}
	return ""
}

func (set *Flags) StringSlice(name string) []string {
	names := set.searchFS(name)
	for _, n := range names {
		f := set.FlagSet.Lookup(n)
		if f != nil {
			parsed := f.Value.(*cli.StringSlice)
			return parsed.Value()
		}
	}
	return nil
}

func (set *Flags) Bool(name string) bool {
	names := set.searchFS(name)
	for _, n := range names {
		f := set.FlagSet.Lookup(n)
		if f != nil {
			parsed, _ := strconv.ParseBool(f.Value.String())
			if parsed {
				return parsed
			}
		}
	}
	return false
}

func (set *Flags) Int(name string) int {
	names := set.searchFS(name)
	for _, n := range names {
		f := set.FlagSet.Lookup(n)
		if f != nil {
			parsed, err := strconv.ParseInt(f.Value.String(), 0, 64)
			if err != nil {
				return int(parsed)
			}
		}
	}
	return 0
}

func (set *Flags) Int64(name string) int64 {
	names := set.searchFS(name)
	for _, n := range names {
		f := set.FlagSet.Lookup(n)
		if f != nil {
			parsed, err := strconv.ParseInt(f.Value.String(), 0, 64)
			if err != nil {
				return parsed
			}
		}
	}
	return 0
}
