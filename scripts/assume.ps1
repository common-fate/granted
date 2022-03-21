#ASSUME - a powershell script to assume an AWS IAM role from the command-line

#ASSUME_FLAG - what assumego told the shell to do
#ASSUME_n - the data from assumego
$env:SHELL="ps"
$env:GRANTED_ALIAS_CONFIGURED="true"
$ASSUME_FLAG, $ASSUME_OUTPUT = `
$(assumego $args) -split '\s+'
$env:ASSUME_STATUS = $LASTEXITCODE

$ASSUME_1, $ASSUME_2, $ASSUME_3, $ASSUME_4, $ASSUME_5 = `
$ASSUME_OUTPUT -split '\s+'


if ( $ASSUME_FLAG -eq "GrantedDesume" ) {
    $env:AWS_ACCESS_KEY_ID = ""
    $env:AWS_SECRET_ACCESS_KEY = ""
    $env:AWS_SESSION_TOKEN = ""
    $env:AWS_REGION = ""
    $env:AWS_PROFILE = ""
    exit
}

#ASSUME the profile
elseif ( $ASSUME_FLAG -eq "GrantedAssume") {
    #Remove the environment variables associated with the AWS CLI,
    #ensuring all environment variables will be valid
    $env:AWS_ACCESS_KEY_ID = ""
    $env:AWS_SECRET_ACCESS_KEY = ""
    $env:AWS_SESSION_TOKEN = ""
    $env:AWS_REGION = ""
    $env:AWS_PROFILE = ""

    $env:ASSUME_COMMAND=$args
    if ( $ASSUME_1 -ne "None" ) {
        $env:AWS_ACCESS_KEY_ID = $ASSUME_1
    }
    if ( $ASSUME_2 -ne "None" ) {
        $env:AWS_SECRET_ACCESS_KEY = $ASSUME_2
    }

    if ( $ASSUME_3 -ne "None" ) {
        $env:AWS_SESSION_TOKEN = $ASSUME_3
    }

    if ( $ASSUME_5 -ne "None" ) {
        $env:ASSUME_PROFILE = $ASSUME_5
    }
    
    if ( $ASSUME_4 -ne "None" ) {
        $env:AWS_REGION = $ASSUME_4
    }
}

elseif ( $ASSUME_FLAG -eq "GrantedOutput") {
    Write-Host "$ASSUME_OUTPUT"
}

exit $env:ASSUME_STATUS