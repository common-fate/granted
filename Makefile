assume-binary:
	go build -o ./bin/dassumego cmd/assume/main.go

cli: assume-binary
	mv ./bin/dassumego /usr/local/bin/
	# replace references to "assumego" (the production binary) with "dassumego"
	cat scripts/assume | sed 's/assumego/dassumego/g' > /usr/local/bin/dassume && chmod +x /usr/local/bin/dassume
	cat scripts/assume.fish | sed 's/assumego/dassumego/g' > /usr/local/bin/dassume.fish && chmod +x /usr/local/bin/dassume.fish

clean:
	rm /usr/local/bin/dassumego
	rm /usr/local/bin/dassume
	rm /usr/local/bin/dassume.fish