
go-binary:
	go build -o ./bin/dgranted cmd/granted/main.go

assume-binary:
	go build -o ./bin/dassumego cmd/assume/main.go

cli: go-binary assume-binary
	mv ./bin/dgranted /usr/local/bin/
	mv ./bin/dassumego /usr/local/bin/
	# replace references to "assumego" (the production binary) with "dassumego"
	cat scripts/assume | sed 's/assumego/dassumego/g' > /usr/local/bin/dassume && chmod +x /usr/local/bin/dassume
	cat scripts/assume.fish | sed 's/assumego/dassumego/g' > /usr/local/bin/dassume.fish && chmod +x /usr/local/bin/dassume.fish

clean:
	rm /usr/local/bin/dassumego
	rm /usr/local/bin/dassume
	rm /usr/local/bin/dassume.fish

aws-credentials: 
	echo -e "\nAWS_ACCESS_KEY_ID=\"$$AWS_ACCESS_KEY_ID\"\nAWS_SECRET_ACCESS_KEY=\"$$AWS_SECRET_ACCESS_KEY\"\nAWS_SESSION_TOKEN=\"$$AWS_SESSION_TOKEN\"\nAWS_REGION=\"$$AWS_REGION\""

test-browser-binary:
	go build -o ./bin/tbrowser cmd/test_browser/main.go
	chmod +x ./bin/tbrowser
	chmod +x ./scripts/test