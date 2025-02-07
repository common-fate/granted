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
	Profile     string
	Region      string
	Service     []string
	Destination string
}

// awsSession is the JSON payload sent to AWS
// to exchange an AWS session for a console URL.
type awsSession struct {
	// SessionID maps to AWS Access Key ID
	SessionID string `json:"sessionId"`
	// SessionKey maps to AWS Secret Access Key
	SessionKey string `json:"sessionKey"`
	// SessionToken maps to AWS Session Token
	SessionToken string `json:"sessionToken"`
}

func (a AWS) URLs(creds aws.Credentials) ([]string, error) {

	urls := []string{}

	if a.Region == "" && a.Service == nil {
		url, err := a.URL(creds, "", "")
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
		return urls, nil
	}

	if len(a.Service) > 0 {
		var region string
		if len(a.Region) > 0 {
			region = a.Region
		}
		for _, service := range a.Service {
			url, err := a.URL(creds, region, service)
			if err != nil {
				return nil, err
			}
			urls = append(urls, url)
		}
	}

	return urls, nil
}

// URL retrieves an authorised access URL for the AWS console. The URL includes a security token which is retrieved
// by exchanging AWS session credentials using the AWS federation endpoint.
//
// see: https://docs.aws.amazon.com/IAM/latest/UserGuide/example_sts_Scenario_ConstructFederatedUrl_section.html
func (a AWS) URL(creds aws.Credentials, region string, service string) (string, error) {
	sess := awsSession{
		SessionID:    creds.AccessKeyID,
		SessionKey:   creds.SecretAccessKey,
		SessionToken: creds.SessionToken,
	}
	sessJSON, err := json.Marshal(sess)
	if err != nil {
		return "", err
	}

	partition := GetPartitionFromRegion(region)
	clio.Debugf("Partition is detected as %s for region %s...\n", partition, a.Region)

	u := url.URL{
		Scheme: "https",
		Host:   partition.RegionalHostString(region),
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
		Host:   partition.RegionalHostString(region),
		Path:   "/federation",
	}

	dest, err := makeDestinationURL(service, region, a.Destination)

	if err != nil {
		return "", err
	}
	q = u.Query()
	q.Add("Action", "login")
	q.Add("Issuer", "")
	q.Add("SigninToken", token.SigninToken)
	q.Add("Destination", dest)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func makeDestinationURL(service string, region string, destination string) (string, error) {
	// if destination is provided, use it
	if destination != "" {
		return destination, nil
	}
	partition := GetPartitionFromRegion(region)
	prefix := partition.RegionalConsoleHostString(region)
	if ServiceMap[service] == "" {
		clio.Warnf("We don't recognize service %s but we'll try and open it anyway (you may receive a 404 page)\n", service)
	} else {
		service = ServiceMap[service]
	}
	dest := prefix + service + "/home"

	// excluding region here if the service is a part of the global service list
	// incomplete list of global services
	_, global := globalServiceMap[service]
	hasRegion := region != ""
	if !global && hasRegion {
		dest = dest + "?region=" + region

	}
	return dest, nil
}
