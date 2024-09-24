package shells

import (
	"fmt"
	"os"
	"strings"
)

// AppendLine writes a line to a file if it does not already exist
func AppendLine(file string, line string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// return an error if the file already contains this line
	if strings.Contains(string(b), line) {
		return &ErrLineAlreadyExists{File: file}
	}

	// open the file for writing
	out, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	// include newlines around the line
	a := fmt.Sprintf("\n%s\n", line)
	_, err = out.WriteString(a)
	if err != nil {
		return err
	}
	return nil
}

// RemoveLine removes a line from a file if it exists
func RemoveLine(file string, lineToRemove string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// the line number in the file where the alias was found
	var lineIndex int
	var found bool

	var ignored []string

	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		removeLine := strings.Contains(line, lineToRemove)

		// When removing the line, if the line after is empty we
		// remove that too. This prevents the length of the config file growing by 1 with blank lines
		// every time Granted adds or removes a line. Really only useful as a nice
		// convenience for developing the Granted CLI, when we do a lot adding and removing lines.
		if found && i == lineIndex+1 && line == "" {
			removeLine = true
		}

		if !removeLine {
			ignored = append(ignored, line)
		} else {
			// mark that we've found the line in the file
			found = true
			lineIndex = i
		}
	}

	if !found {
		// we didn't find the line in the file, so return an error in order to let the user know that it doesn't exist there.
		return &ErrLineNotFound{File: file}
	}

	output := strings.Join(ignored, "\n")
	err = os.WriteFile(file, []byte(output), 0644)
	if err != nil {
		return err
	}
	return nil
}
