package console

import "strings"

type PartitionHost int

const (
	Default PartitionHost = iota
	Gov
	Cn
	ISO
	ISOB
)

func (p PartitionHost) String() string {
	switch p {
	case Default:
		return "aws"
	case Gov:
		return "aws-us-gov"
	case Cn:
		return "aws-cn"
	case ISO:
		return "aws-iso"
	case ISOB:
		return "aws-iso-b"
	}
	return "aws"
}

func (p PartitionHost) HostString() string {
	switch p {
	case Default:
		return "signin.aws.amazon.com"
	case Gov:
		return "signin.amazonaws-us-gov.com"
	case Cn:
		return "signin.amazonaws.cn"
	}
	// Note: we're not handling the ISO and ISOB cases, I don't think they are supported by a public AWS console
	return "signin.aws.amazon.com"
}

func (p PartitionHost) ConsoleHostString() string {
	switch p {
	case Default:
		return "https://console.aws.amazon.com/"
	case Gov:
		return "https://console.amazonaws-us-gov.com/"
	case Cn:
		return "https://console.amazonaws.cn/"
	}
	// Note: we're not handling the ISO and ISOB cases, I don't think they are supported by a public AWS console
	return "https://console.aws.amazon.com/"
}

func GetPartitionFromRegion(region string) PartitionHost {
	partition := strings.Split(region, "-")
	if partition[0] == "cn" {
		return PartitionHost(Cn)
	}
	if len(partition) > 1 {
		if partition[1] == "iso" {
			return PartitionHost(ISO)
		}
		if partition[1] == "isob" {
			return PartitionHost(ISOB)
		}
		if partition[1] == "gov" {
			return PartitionHost(Gov)
		}
	}
	return PartitionHost(Default)
}
