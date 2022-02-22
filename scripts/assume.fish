#!/bin/fish

set -gx GRANTED_ALIAS_CONFIGURED "true"

#GRANTED_FLAG - what granted told the shell to do
#GRANTED_n - the data from granted
set GRANTED_OUTPUT (assumego $argv)
set GRANTED_STATUS $status
echo $GRANTED_OUTPUT | read GRANTED_FLAG GRANTED_1 GRANTED_2 GRANTED_3 GRANTED_4


# remove carriage return
set -gx GRANTED_FLAG (echo $GRANTED_FLAG | tr -d '\r')

if test "$GRANTED_FLAG" = "NAME:"
  assumego $argv

else if test "$GRANTED_FLAG" = "GrantedAssume"
  set -e AWS_ACCESS_KEY_ID
  set -e AWS_SECRET_ACCESS_KEY
  set -e AWS_SESSION_TOKEN
  set -e GRANTED_AWS_ROLE_PROFILE

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
    set -gx GRANTED_AWS_ROLE_PROFILE $GRANTED_4
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
      echo set -gx GRANTED_AWS_ROLE_PROFILE $GRANTED_4
    end
  end
end

exit $GRANTED_STATUS