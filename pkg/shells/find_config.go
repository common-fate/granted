package shells

import (
	"os"
	"path"
)

func GetFishConfigFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	file := path.Join(home, ".config", "fish", "config.fish")

	// check that the file exists; create it if not
	if _, err := os.Stat(file); os.IsNotExist(err) {
		f, err := os.Create(file)
		if err != nil {
			return "", err
		}
		defer f.Close()
	}
	return file, nil
}

func GetBashConfigFile() (string, error) {

	bashLoginFiles := []string{
		".bash_profile",
		".bash_login",
		".profile",
		".bashrc",
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	for _, f := range bashLoginFiles {
		file := path.Join(home, f)
		if _, err := os.Stat(file); err == nil {

			return file, nil
		}
	}
	// if we got here, none of the bash login files we tried above work
	// so use the .bash_profile
	file := path.Join(home, ".bash_profile")

	// check that the file exists; create it if not
	if _, err := os.Stat(file); os.IsNotExist(err) {
		f, err := os.Create(file)
		if err != nil {
			return "", err
		}
		defer f.Close()
	}

	return file, nil
}

func GetZshConfigFile() (string, error) {
	// ZDOTDIR is used by ZSH for loading config
	dir := os.Getenv("ZDOTDIR")

	if dir == "" {
		// fallback to the home directory if ZDOTDIR isn't set
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = home
	}

	file := path.Join(dir, ".zshenv")

	// check that the file exists; create it if not
	if _, err := os.Stat(file); os.IsNotExist(err) {
		f, err := os.Create(file)
		if err != nil {
			return "", err
		}
		defer f.Close()
	}
	return file, nil
}
