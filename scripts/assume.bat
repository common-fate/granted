@echo off
set SHELL=cmd

set GRANTED_ALIAS_CONFIGURED=true
assumego %* 1> %TEMP%\temp-assume.txt
set ASSUME_STATUS=%ERRORLEVEL%
set /p ASSUME_OUTPUT=<%TEMP%\temp-assume.txt
del %TEMP%\temp-assume.txt

@echo off
for /f "tokens=1,2,3,4,5,6,7 delims= " %%a in ("%ASSUME_OUTPUT%") do (
    
    if "%%a" == "GrantedDesume" (
		set AWS_ACCESS_KEY_ID=
		set AWS_SECRET_ACCESS_KEY=
		set AWS_SESSION_TOKEN=
		set GRANTED_AWS_ROLE_PROFILE=
		set AWS_REGION=
        set AWS_SESSION_EXPIRATION=
        Exit /b %ASSUME_STATUS%
    )
    
    if "%%a" == "GrantedAssume" (
		set AWS_ACCESS_KEY_ID=
		set AWS_SECRET_ACCESS_KEY=
		set AWS_SESSION_TOKEN=
		set GRANTED_AWS_ROLE_PROFILE=
		set AWS_REGION=
        set AWS_SESSION_EXPIRATION=

        if "%%b" NEQ "None" (
            set AWS_ACCESS_KEY_ID=%%b)

        if "%%c" NEQ "None" (
            set AWS_SECRET_ACCESS_KEY=%%c)

        if "%%d" NEQ "None" (
            set AWS_SESSION_TOKEN=%%d)

        if "%%e" NEQ "None" (
            set AWS_PROFILE=%%e)
			
        if "%%f" NEQ "None" (
            set AWS_REGION=%%f)

        if "%%g" NEQ "None" (
            set AWS_SESSION_EXPIRATION=%%g)

        Exit /b %ASSUME_STATUS%
    )
    echo %ASSUME_OUTPUT%
)
