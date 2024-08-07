#!/bin/tcsh

#this is set to true because the alias will be configured to point to the tcsh script in a previous step
#this happens in the assume script
setenv GRANTED_ALIAS_CONFIGURED "true"

set GRANTED_OUTPUT = `assumego $argv`
set GRANTED_STATUS = $status
set GRANTED_FLAG = $GRANTED_OUTPUT[1]

if ( "$GRANTED_FLAG" == "NAME:" ) then
  assumego $argv
else if ( "$GRANTED_FLAG" == "GrantedDesume" ) then
  unsetenv AWS_ACCESS_KEY_ID
  unsetenv AWS_SECRET_ACCESS_KEY
  unsetenv AWS_SESSION_TOKEN
  unsetenv AWS_PROFILE
  unsetenv AWS_REGION
  unsetenv AWS_DEFAULT_REGION
  unsetenv AWS_SESSION_EXPIRATION
  unsetenv AWS_CREDENTIAL_EXPIRATION
  unsetenv GRANTED_SSO
  unsetenv GRANTED_SSO_START_URL
  unsetenv GRANTED_SSO_ROLE_NAME
  unsetenv GRANTED_SSO_REGION
  unsetenv GRANTED_SSO_ACCOUNT_ID
else if ( "$GRANTED_FLAG" == "GrantedAssume" ) then
  unsetenv AWS_ACCESS_KEY_ID
  unsetenv AWS_SECRET_ACCESS_KEY
  unsetenv AWS_SESSION_TOKEN
  unsetenv AWS_PROFILE
  unsetenv AWS_REGION
  unsetenv AWS_DEFAULT_REGION
  unsetenv AWS_SESSION_EXPIRATION
  unsetenv AWS_CREDENTIAL_EXPIRATION
  unsetenv GRANTED_SSO
  unsetenv GRANTED_SSO_START_URL
  unsetenv GRANTED_SSO_ROLE_NAME
  unsetenv GRANTED_SSO_REGION
  unsetenv GRANTED_SSO_ACCOUNT_ID

  setenv GRANTED_COMMAND $argv
  if ( "$GRANTED_OUTPUT[2]" != "None" ) then
    setenv AWS_ACCESS_KEY_ID $GRANTED_OUTPUT[2]
  endif
  if ( "$GRANTED_OUTPUT[3]" != "None" ) then
    setenv AWS_SECRET_ACCESS_KEY $GRANTED_OUTPUT[3]
  endif
  if ( "$GRANTED_OUTPUT[4]" != "None" ) then
    setenv AWS_SESSION_TOKEN $GRANTED_OUTPUT[4]
  endif
  if ( "$GRANTED_OUTPUT[5]" != "None" ) then
    setenv AWS_PROFILE $GRANTED_OUTPUT[5]
  endif
  if ( "$GRANTED_OUTPUT[6]" != "None" ) then
    setenv AWS_REGION $GRANTED_OUTPUT[6]
    setenv AWS_DEFAULT_REGION $GRANTED_OUTPUT[6]
  endif
  if ( "$GRANTED_OUTPUT[7]" != "None" ) then
    setenv AWS_SESSION_EXPIRATION $GRANTED_OUTPUT[7]
    setenv AWS_CREDENTIAL_EXPIRATION $GRANTED_OUTPUT[7]
  endif
  if ( "$GRANTED_OUTPUT[8]" != "None" ) then
    setenv GRANTED_SSO $GRANTED_OUTPUT[8]
  endif
  if ( "$GRANTED_OUTPUT[9]" != "None" ) then
    setenv GRANTED_SSO_START_URL $GRANTED_OUTPUT[9]
  endif
  if ( "$GRANTED_OUTPUT[10]" != "None" ) then
    setenv GRANTED_SSO_ROLE_NAME $GRANTED_OUTPUT[10]
  endif
  if ( "$GRANTED_OUTPUT[11]" != "None" ) then
    setenv GRANTED_SSO_REGION $GRANTED_OUTPUT[11]
  endif
  if ( "$GRANTED_OUTPUT[12]" != "None" ) then
    setenv GRANTED_SSO_ACCOUNT_ID $GRANTED_OUTPUT[12]
  endif

else if ( "$GRANTED_FLAG" == "GrantedOutput" ) then
  foreach line ($GRANTED_OUTPUT)
      if ( "$line" != "GrantedOutput" ) then
        echo $line
      endif
  end


else if ( "$GRANTED_FLAG" == "GrantedExec" ) then
  # Set GRANTED_OUTPUT[13] with a command to execute, for example, "bash -c 'some_command'"
  # Make sure to properly escape quotes if needed and pass arguments.

  # Set and export the AWS variables only for the duration of the 'sh -c' command
  env AWS_ACCESS_KEY_ID=$GRANTED_OUTPUT[2] \
    AWS_SECRET_ACCESS_KEY=$GRANTED_OUTPUT[3] \
    AWS_SESSION_TOKEN=$GRANTED_OUTPUT[4] \
    AWS_REGION=$GRANTED_OUTPUT[6] \
    AWS_DEFAULT_REGION=$GRANTED_OUTPUT[6] \
    sh -c "$GRANTED_OUTPUT[13]"
endif

exit $GRANTED_STATUS