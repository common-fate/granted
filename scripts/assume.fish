#!/bin/fish

#this is set to true because the alias will be configured to point to the fish script in a previous step
#this happens in the assume script
set -gx GRANTED_ALIAS_CONFIGURED "true"

#GRANTED_FLAG - what granted told the shell to do
#GRANTED_n - the data from granted

set GRANTED_OUTPUT (assumego $argv)
set GRANTED_STATUS $status
echo $GRANTED_OUTPUT | IFS=' ' read GRANTED_FLAG GRANTED_1 GRANTED_2 GRANTED_3 GRANTED_4 GRANTED_5 GRANTED_6 GRANTED_7 GRANTED_8 GRANTED_9 GRANTED_10 GRANTED_11


# remove carriage return
set -gx GRANTED_FLAG (echo $GRANTED_FLAG | tr -d '\r')

if test "$GRANTED_FLAG" = "NAME:"
  assumego $argv
else if test "$GRANTED_FLAG" = "GrantedDesume"
  set -e AWS_ACCESS_KEY_ID
  set -e AWS_SECRET_ACCESS_KEY
  set -e AWS_SESSION_TOKEN
  set -e AWS_PROFILE
  set -e AWS_REGION
  set -e AWS_SESSION_EXPIRATION
  set -e AWS_CREDENTIAL_EXPIRATION
  set -e GRANTED_SSO
  set -e GRANTED_SSO_START_URL
  set -e GRANTED_SSO_ROLE_NAME
  set -e GRANTED_SSO_REGION
  set -e GRANTED_SSO_ACCOUNT_ID
else if test "$GRANTED_FLAG" = "GrantedAssume"
  set -e AWS_ACCESS_KEY_ID
  set -e AWS_SECRET_ACCESS_KEY
  set -e AWS_SESSION_TOKEN
  set -e AWS_PROFILE
  set -e AWS_REGION
  set -e AWS_SESSION_EXPIRATION
  set -e AWS_CREDENTIAL_EXPIRATION
  set -e GRANTED_SSO
  set -e GRANTED_SSO_START_URL
  set -e GRANTED_SSO_ROLE_NAME
  set -e GRANTED_SSO_REGION
  set -e GRANTED_SSO_ACCOUNT_ID

  set -gx GRANTED_COMMAND $argv
  if test "$GRANTED_1" != "None"
    set -gx AWS_ACCESS_KEY_ID $GRANTED_1
  end
  if test "$GRANTED_2" != "None"
    set -gx AWS_SECRET_ACCESS_KEY $GRANTED_2
  end
  if test "$GRANTED_3" != "None"
    set -gx AWS_SESSION_TOKEN $GRANTED_3
  end
  if test "$GRANTED_4" != "None"
    set -gx AWS_PROFILE $GRANTED_4
  end
  if test "$GRANTED_5" != "None"
    set -gx AWS_REGION $GRANTED_5
  end
  if test "$GRANTED_6" != "None"
    set -gx AWS_SESSION_EXPIRATION $GRANTED_6
    set -gx AWS_CREDENTIAL_EXPIRATION $GRANTED_6
  end
  if test "$GRANTED_7" != "None"
    set -gx GRANTED_SSO $GRANTED_7
  end
  if test "$GRANTED_8" != "None"
    set -gx GRANTED_SSO_START_URL $GRANTED_8
  end
  if test "$GRANTED_9" != "None"
    set -gx GRANTED_SSO_ROLE_NAME $GRANTED_9
  end
  if test "$GRANTED_10" != "None"
    set -gx GRANTED_SSO_REGION $GRANTED_10
  end
  if test "$GRANTED_11" != "None"
    set -gx GRANTED_SSO_ACCOUNT_ID $GRANTED_11
  end

  if contains -- -s $argv
    if test "$GRANTED_1" != "None"
      echo set -gx AWS_ACCESS_KEY_ID $GRANTED_1
    end
    if test "$GRANTED_2" != "None"
      echo set -gx AWS_SECRET_ACCESS_KEY $GRANTED_2
    end
    if test "$GRANTED_3" != "None"
      echo set -gx AWS_SESSION_TOKEN $GRANTED_3
    end
    if test "$GRANTED_4" != "None"
      echo set -gx AWS_PROFILE $GRANTED_4
    end
    if test "$GRANTED_5" != "None"
      echo set -gx AWS_REGION $GRANTED_5
    end
    if test "$GRANTED_6" != "None"
      echo set -gx AWS_SESSION_EXPIRATION $GRANTED_6
      echo set -gx AWS_CREDENTIAL_EXPIRATION $GRANTED_6
    end
  end

else if test "$GRANTED_FLAG" = "GrantedOutput"
  for line in $GRANTED_OUTPUT
      if test "$line" != "GrantedOutput"
        echo $line
      end
  end
end

exit $GRANTED_STATUS