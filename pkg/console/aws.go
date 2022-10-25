package console

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/clio"
)

type AWS struct {
	Profile string
	Region  string
	Service string
}

// awsSession is the JSON payload sent to AWS
// to exchange an AWS session for a console URL.
type awsSession struct {
	// SessionID maps to AWS Access Key ID
	SessionID string `json:"sessionId"`
	// SessionKey maps to AWS Secret Access Key
	SesssionKey string `json:"sessionKey"`
	// SessionToken maps to AWS Session Token
	SessionToken string `json:"sessionToken"`
}

// URL retrieves an authorised access URL for the AWS console. The URL includes a security token which is retrieved
// by exchanging AWS session credentials using the AWS federation endpoint.
//
// see: https://docs.aws.amazon.com/IAM/latest/UserGuide/example_sts_Scenario_ConstructFederatedUrl_section.html
func (a AWS) URL(creds aws.Credentials) (string, error) {
	sess := awsSession{
		SessionID:    creds.AccessKeyID,
		SesssionKey:  creds.SecretAccessKey,
		SessionToken: creds.SessionToken,
	}
	sessJSON, err := json.Marshal(sess)
	if err != nil {
		return "", err
	}

	partition := GetPartitionFromRegion(a.Region)
	clio.Debug("Partition is detected as %s for region %s...\n", partition, a.Region)

	u := url.URL{
		Scheme: "https",
		Host:   partition.HostString(),
		Path:   "/federation",
	}
	q := u.Query()
	q.Add("Action", "getSigninToken")
	q.Add("Session", string(sessJSON))
	u.RawQuery = q.Encode()

	res, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("opening console failed with code %v", res.StatusCode)
	}

	token := struct {
		SigninToken string `json:"SigninToken"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		return "", err
	}

	u = url.URL{
		Scheme: "https",
		Host:   partition.HostString(),
		Path:   "/federation",
	}

	dest, err := makeDestinationURL(a.Service, a.Region)

	if err != nil {
		return "", err
	}
	q = u.Query()
	q.Add("Action", "login")
	q.Add("Issuer", "")
	q.Add("Destination", dest)
	q.Add("SigninToken", token.SigninToken)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func makeDestinationURL(service string, region string) (string, error) {
	partition := GetPartitionFromRegion(region)
	prefix := partition.ConsoleHostString()
	if ServiceMap[service] == "" {
		clio.Warn("We don't recognize service %s but we'll try and open it anyway (you may receive a 404 page)\n", service)
	} else {
		service = ServiceMap[service]
	}
	dest := prefix + service + "/home"

	//excluding region here if the service is apart of the global service list
	//incomplete list of global services
	_, global := globalServiceMap[service]
	hasRegion := region != ""
	if !global && hasRegion {
		dest = dest + "?region=" + region

	}
	return dest, nil
}
