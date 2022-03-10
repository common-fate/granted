package browsers

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/common-fate/granted/pkg/debug"
	"github.com/fatih/color"
)

type plist struct {
	//XMLName xml.Name `xml:"plist"`
	Pdict Pdict `xml:"dict"`
}

type Pdict struct {
	//XMLName xml.Name `xml:"dict"`
	Key   string `xml:"key"`
	Array Array  `xml:"array"`
}

type Array struct {
	//XMLName xml.Name `xml:"array"`
	Dict Dict `xml:"dict"`
}

type Dict struct {
	//XMLName xml.Name `xml:"dict"`
	Key     []string `xml:"key"`
	Dict    IntDict  `xml:"dict"`
	Strings []string `xml:"string"`
}

type IntDict struct {
	//XMLName xml.Name `xml:"dict"`
	Key     string `xml:"key"`
	Strings string `xml:"string"`
}

func HandleOSXBrowserSearch() (string, error) {
	//get home dir
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := home + "/Library/Preferences/com.apple.LaunchServices/com.apple.launchservices.secure.plist"

	//convert plist to xml using putil
	//plutil -convert xml1
	args := []string{"-convert", "xml1", path}
	cmd := exec.Command("plutil", args...)
	err = cmd.Run()
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, color.Error, err.Error())
	}

	//read plist file
	data, err := ioutil.ReadFile(path)

	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, color.Error, err.Error())
	}
	plist := &plist{}

	// fmt.Fprintf(color.Error, "\n%s\n", data)
	//unmarshal the xml into the structs
	err = xml.Unmarshal([]byte(data), &plist)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, color.Error, err.Error())
	}

	//get out the default browser

	for i, s := range plist.Pdict.Array.Dict.Strings {
		if s == "http" {
			return plist.Pdict.Array.Dict.Strings[i-1], nil
		}
	}
	return "", nil
}
