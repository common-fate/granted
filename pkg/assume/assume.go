package assume

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

func AssumeCommand(c *cli.Context) error {

	role := "rolename goes here"
	account := "123456789120"
	accessKeyID := "todo"
	secretAccessKey := "todo"
	sessionToken := "todo"
	expiration := time.Now().Add(time.Hour)

	sess := Session{SessionID: accessKeyID, SesssionKey: secretAccessKey, SessionToken: sessionToken}
	labels := RoleLabels{Role: role, Account: account}
	if c.Bool("console") {
		return LaunchConsoleSession(sess, labels, BrowserDefault)
	} else if c.Bool("extension") {
		return LaunchConsoleSession(sess, labels, BrowerFirefox)
	} else if c.Bool("chrome") {
		return LaunchConsoleSession(sess, labels, BrowserChrome)
	} else {
		fmt.Printf("GrantedAssume %s %s %s", accessKeyID, secretAccessKey, sessionToken)
		fmt.Fprintf(os.Stderr, "\033[32m[%s] session credentials will expire %s\033[0m\n", role, expiration.Local().String())
	}

	return nil
}
