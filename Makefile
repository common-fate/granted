PREFIX?=/usr/local

go-binary:
	go build -o ./bin/dgranted cmd/granted/main.go

assume-binary:
	go build -o ./bin/dassumego cmd/assume/main.go

cli: go-binary assume-binary
	mv ./bin/dgranted ${PREFIX}/bin/granted
	mv ./bin/dassumego ${PREFIX}/bin/
	# replace references to "assumego" (the production binary) with "dassumego"
	cat scripts/assume | sed 's/assumego/dassumego/g' > ${PREFIX}/bin/dassume && chmod +x ${PREFIX}/bin/dassume
	cat scripts/assume.fish | sed 's/assumego/dassumego/g' > ${PREFIX}/bin/dassume.fish && chmod +x ${PREFIX}/bin/dassume.fish
	mv ${PREFIX}/bin/dassume ${PREFIX}/bin/assume
	mv ${PREFIX}/bin/dassume.fish ${PREFIX}/bin/assume.fish

clean:
	rm ${PREFIX}/bin/dassumego
	rm ${PREFIX}/bin/dassume
	rm ${PREFIX}/bin/dassume.fish

aws-credentials: 
	echo -e "\nAWS_ACCESS_KEY_ID=\"$$AWS_ACCESS_KEY_ID\"\nAWS_SECRET_ACCESS_KEY=\"$$AWS_SECRET_ACCESS_KEY\"\nAWS_SESSION_TOKEN=\"$$AWS_SESSION_TOKEN\"\nAWS_REGION=\"$$AWS_REGION\""

test-browser-binary:
	GOOS=linux go build -o ./bin/linux/tbrowser cmd/testing/browser/main.go
	GOOS=darwin GOARCH=amd64 go build -o ./bin/macos/tbrowser cmd/testing/browser/main.go
	GOOS=windows go build -o ./bin/windows/tbrowser.exe cmd/testing/browser/main.go

test-creds-binary:
	GOOS=linux go build -o ./bin/linux/tcreds cmd/testing/creds/main.go
	GOOS=darwin GOARCH=amd64 go build -o ./bin/macos/tcreds cmd/testing/creds/main.go
	GOOS=windows go build -o ./bin/windows/tcreds.exe cmd/testing/creds/main.go

test-binaries: test-browser-binary test-creds-binary


ci-cli-all-platforms: test-binaries
	# build steps
	GOOS=linux go build -o ./bin/linux/dgranted cmd/granted/main.go
	GOOS=darwin GOARCH=amd64 go build -o ./bin/macos/dgranted cmd/granted/main.go
	GOOS=windows go build -o ./bin/windows/dgranted.exe cmd/granted/main.go
	GOOS=linux go build -o ./bin/linux/dassumego cmd/assume/main.go
	GOOS=darwin GOARCH=amd64 go build -o ./bin/macos/dassumego cmd/assume/main.go
	GOOS=windows go build -o ./bin/windows/dassumego.exe cmd/assume/main.go
	# replace references to "assumego" (the production binary) with "dassumego"
	cat scripts/assume | sed 's/assumego/dassumego/g' > bin/linux/dassume && chmod +x bin/linux/dassume
	cat scripts/assume.fish | sed 's/assumego/dassumego/g' > bin/linux/dassume.fish && chmod +x bin/linux/dassume.fish
	cp bin/linux/dassume bin/macos/dassume
	cp bin/linux/dassume.fish bin/macos/dassume.fish
	cat scripts/assume.bat | sed 's/assumego/dassumego/g' > bin/windows/dassume.bat
	cat scripts/assume.ps1 | sed 's/assumego/dassumego/g' > bin/windows/dassume.ps1