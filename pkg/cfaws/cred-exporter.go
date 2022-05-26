package cfaws

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/fatih/color"
)

// ExportCredsToProfile will write assumed credentials to ~/.aws/credentials with a specified profile name header
func ExportCredsToProfile(profileName string, creds aws.Credentials) error {
	// fetch the parsed cred file
	credPath := config.DefaultSharedCredentialsFilename()
	// credPath := "./test-creds"

	//create it if it doesn't exist
	if _, err := os.Stat(credPath); os.IsNotExist(err) {

		f, err := os.Create(credPath)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
		fmt.Fprintln(color.Error, "Created file.")

	}

	//Check to see if there already is a cred profile for this profile
	input, err := ioutil.ReadFile(credPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	found := false

	//might need to bundle this up in a goroutine if it becomes slow for large cred files
	for i, line := range lines {
		//replace creds if exist
		if strings.Contains(line, "["+profileName+"]") {
			found = true

			lines[i] = "[" + profileName + "]"
			lines[i+1] = "aws_access_key_id=" + creds.AccessKeyID
			lines[i+2] = "aws_secret_access_key=" + creds.SecretAccessKey
			lines[i+3] = "aws_session_token=" + creds.SessionToken

			break
		}
	}

	// //otherwise just write new profile
	// //append the new profile to the creds
	if !found {
		lines = append(lines, "["+profileName+"]")
		lines = append(lines, "aws_access_key_id="+creds.AccessKeyID)
		lines = append(lines, "aws_secret_access_key="+creds.SecretAccessKey)
		lines = append(lines, "aws_session_token="+creds.SessionToken)
	}
	//put the file back together
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(credPath, []byte(output), 0644)
	if err != nil {
		log.Fatalln(err)
	}

	return nil
}
