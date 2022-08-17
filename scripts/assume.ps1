#ASSUME - a powershell script to assume an AWS IAM role from the command-line

#ASSUME_FLAG - what assumego told the shell to do
#ASSUME_n - the data from assumego
$env:SHELL="ps"
$env:GRANTED_ALIAS_CONFIGURED="true"
$ASSUME_FLAG, $ASSUME_1, $ASSUME_2, $ASSUME_3, $ASSUME_4, $ASSUME_5, $ASSUME_6= `
$(& (Join-Path $PSScriptRoot -ChildPath "assumego") $args) -split '\s+'
$env:ASSUME_STATUS = $LASTEXITCODE


if ( $ASSUME_FLAG -eq "GrantedDesume" ) {
    $env:AWS_ACCESS_KEY_ID = ""
    $env:AWS_SECRET_ACCESS_KEY = ""
    $env:AWS_SESSION_TOKEN = ""
    $env:AWS_PROFILE = ""
    $env:AWS_REGION = ""
    $env:AWS_SESSION_EXPIRATION = ""
    exit
}

#ASSUME the profile
elseif ( $ASSUME_FLAG -eq "GrantedAssume") {
    #Remove the environment variables associated with the AWS CLI,
    #ensuring all environment variables will be valid
    $env:AWS_ACCESS_KEY_ID = ""
    $env:AWS_SECRET_ACCESS_KEY = ""
    $env:AWS_SESSION_TOKEN = ""
    $env:AWS_PROFILE = ""
    $env:AWS_REGION = ""
    $env:AWS_SESSION_EXPIRATION = ""

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

    if ( $ASSUME_4 -ne "None" ) {
        $env:AWS_PROFILE = $ASSUME_4
    }
    
    if ( $ASSUME_5 -ne "None" ) {
        $env:AWS_REGION = $ASSUME_5
    }

    if ( $ASSUME_6 -ne "None" ) {
        $env:AWS_SESSION_EXPIRATION = $ASSUME_6
    }
}

#ASSUME the profile
elseif ( $ASSUME_FLAG -eq "GrantedAssumeSSO") {
    #Remove the environment variables associated with the AWS CLI,
    #ensuring all environment variables will be valid
    $env:AWS_ACCESS_KEY_ID = ""
    $env:AWS_SECRET_ACCESS_KEY = ""
    $env:AWS_SESSION_TOKEN = ""
    $env:AWS_PROFILE = ""
    $env:AWS_REGION = ""
    $env:AWS_SESSION_EXPIRATION = ""

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
    if ( $ASSUME_4 -ne "None" ) {
        $env:AWS_REGION = $ASSUME_4
    }

    if ( $ASSUME_5 -ne "None" ) {
        $env:AWS_SESSION_EXPIRATION = $ASSUME_5
    }
}


elseif ( $ASSUME_FLAG -eq "GrantedOutput") {
    Write-Host "$ASSUME_1"
}

exit $env:ASSUME_STATUS
