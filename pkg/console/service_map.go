package console

// ServiceMap maps CLI flags to AWS console URL paths.
// e.g. passing in `-r ec2` will open the console at the ec2/v2 URL.
var ServiceMap = map[string]string{
	"":               "console",
	"ec2":            "ec2/v2",
	"sso":            "singlesignon",
	"ecs":            "ecs",
	"eks":            "eks",
	"athena":         "athena",
	"cloudmap":       "cloudmap",
	"c9":             "cloud9",
	"cfn":            "cloudformation",
	"cloudformation": "cloudformation",
	"cloudwatch":     "cloudwatch",
	"gd":             "guardduty",
	"l":              "lambda",
	"cw":             "cloudwatch",
	"cf":             "cloudfront",
	"ct":             "cloudtrail",
	"ddb":            "dynamodbv2",
	"eb":             "elasticbeanstalk",
	"ebs":            "elasticbeanstalk",
	"ecr":            "ecr",
	"grafana":        "grafana",
	"lambda":         "lambda",
	"route53":        "route53/v2",
	"r53":            "route53/v2",
	"s3":             "s3",
	"secretsmanager": "secretsmanager",
	"iam":            "iamv2",
	"waf":            "wafv2",
	"rds":            "rds",
	"dms":            "dms/v2",
	"mwaa":           "mwaa",
	"param":          "systems-manager/parameters",
	"redshift":       "redshiftv2",
	"sagemaker":      "sagemaker",
	"ssm":            "systems-manager",
}

var globalServiceMap = map[string]bool{
	"iam":     true,
	"route53": true,
	"r53":     true,
}
