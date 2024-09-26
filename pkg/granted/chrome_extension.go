package granted

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/common-fate/granted/internal/build"
	"github.com/common-fate/granted/pkg/chromemsg"
	"github.com/common-fate/granted/pkg/securestorage"
	"github.com/urfave/cli/v2"
)

func HandleChromeExtensionCall(c *cli.Context) error {
	arg := c.Args().First()

	// When called with a chrome extension, the first argument will be the extension ID,
	// in a format like 'chrome-extension://fcipjekpmlpmiikgdecbjbcpmenmceoh'.
	//
	// We need to verify the extension ID matches our list of allowed Chrome extension IDs.

	u, err := url.Parse(arg)
	if err != nil {
		return fmt.Errorf("invalid Chrome Extension URL %q: %w", arg, err)
	}

	if u.Host != build.ChromeExtensionID {
		return fmt.Errorf("Chrome Extension ID %q did not match allowed ID %q", u.Host, build.ChromeExtensionID)
	}

	// if we get here, the Granted CLI has been invoked from our browser extension.
	s := chromemsg.Server{
		Input:  os.Stdin,
		Output: os.Stdout,
	}

	var msg chromeMessage

	err = json.NewDecoder(&s).Decode(&msg)
	if err != nil {
		return err
	}

	if msg.Type != "get_valid_user_codes" {
		return fmt.Errorf("invalid type field: %s", msg.Type)
	}

	storage := securestorage.NewDeviceCodeSecureStorage()
	codes, err := storage.GetValidUserCodes()
	if err != nil {
		return err
	}

	err = json.NewEncoder(&s).Encode(codes)
	if err != nil {
		return err
	}

	return os.Stdout.Sync()
}

type chromeMessage struct {
	Type string `json:"type"`
}
