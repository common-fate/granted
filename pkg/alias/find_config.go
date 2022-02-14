package alias

import (
	"os"
	"path"
)

const fishAlias = `alias assume="source /usr/local/bin/assume.fish"`
const defaultAlias = `alias assume="source assume"`

type Config struct {
	// Alias is the text to insert into the File for setting up the sourcing command for Granted
	Alias string
	File  string
}

func getFishConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	file := path.Join(home, ".config", "fish", "config.fish")

	// check that the file exists; create it if not
	if _, err := os.Stat(file); os.IsNotExist(err) {
		file, err := os.Create(file)
		if err != nil {
			return Config{}, err
		}
		defer file.Close()
	}

	cfg := Config{
		Alias: fishAlias,
		File:  file,
	}
	return cfg, nil
}

func getBashConfig() (Config, error) {
	cfg := Config{
		Alias: defaultAlias,
	}

	bashLoginFiles := []string{
		".bash_profile",
		".bash_login",
		".profile",
		".bashrc",
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	for _, f := range bashLoginFiles {
		file := path.Join(home, f)
		if _, err := os.Stat(file); err == nil {
			cfg.File = file
			return cfg, nil
		}
	}
	// if we got here, none of the bash login files we tried above work
	// so use the .bash_profile
	cfg.File = path.Join(home, ".bash_profile")

	// check that the file exists; create it if not
	if _, err := os.Stat(cfg.File); os.IsNotExist(err) {
		file, err := os.Create(cfg.File)
		if err != nil {
			return Config{}, err
		}
		defer file.Close()
	}

	return cfg, nil
}

func getZshConfig() (Config, error) {
	// ZDOTDIR is used by ZSH for loading config
	dir := os.Getenv("ZDOTDIR")

	if dir == "" {
		// fallback to the home directory if ZDOTDIR isn't set
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		dir = home
	}

	file := path.Join(dir, ".zshenv")

	// check that the file exists; create it if not
	if _, err := os.Stat(file); os.IsNotExist(err) {
		file, err := os.Create(file)
		if err != nil {
			return Config{}, err
		}
		defer file.Close()
	}

	cfg := Config{
		Alias: defaultAlias,
		File:  file,
	}
	return cfg, nil
}
