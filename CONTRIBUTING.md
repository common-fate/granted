# Contributing to granted

We welcome all contributions to Granted. Please read our [Contributor Code of Conduct](./CODE_OF_CONDUCT.md).

## Requirements

The development instructions below pertain to Unix-based systems like Linux and MacOS. If you're running Windows and would like to contribute to Granted, feel free to [reach out to us on Slack](https://join.slack.com/t/commonfatecommunity/shared_invite/zt-q4m96ypu-_gYlRWD3k5rIsaSsqP7QMg) if you're having issues setting up your development environment.

In order to develop Granted you'll need the following:

- [Go 1.17](https://go.dev/doc/install)

## Getting started

Granted consists of two binaries:

- `granted`: used to manage Granted configuration

- `assume`: used to assume roles

You can read about how the `assume` binary exports environment variables [here](https://docs.commonfate.io/granted/internals/shell-alias).

In development we use `dassume` and `dgranted` to avoid collisions between the released and development binaries.

To build the Granted CLI you can run

```
make cli
```

The CLI should now be available on your PATH as `dgranted` and `dassume`.

## Creating a bug report

Receiving bug reports is great, its helps us the identify and patch issues in the application so that the users have as good experience as possible. But it's important to include the necessary information in the bug report so that the maintainers are able to get to the bottom of the problem with as much context as possible.

When opening a bug report we ask that you please include the following information:

- Your Granted version `granted -v`
- If applicable and relevant your .aws config file, found at `~/.aws/config` **(excluding any account IDs or SSO start URLs)**
- Details surrouding the bug and steps to replicate
- If possible an example of the bug

Some things to try before opening a new issue:

- Make sure you're running the latest version of Granted
- Check if there is already an open issue surrounding your bug and add to that open issue

**Example:**

> Short and descriptive example bug report title
>
> Description of the bug
>
> Granted Version: v0.1.5
>
> Your config
>
> ```
> [profile PROFNAME]
> sso_start_url=***
> sso_region=ap-southeast-2
> sso_account_id=***
> sso_role_name=ROLE_NAME
> region=ap-southeast-2
> credential_process=aws-sso-credential-process --profile PROFNAME
> ```
>
> Any other information you want to share that is relevant to the issue being
> reported. This might include the lines of code that you have identified as
> causing the bug, and potential solutions (and your opinions on their
> merits).

# Technical Notes

Before you get started developing on Granted, these notes will help to explain some key comcepts in the codebase.

## IO

Granted consists of 2 binaries, `granted` and `assumego`.
When you run `assume` a shell script will run which wraps the assumego binary.
This is required so that assume can set environment variables in your terminal.

For this reason, output to the terminal from print statements should specify os.Stderr as the writer.
For example

```go
    fmt.Fprintln(os.Stderr, "hello world")
```
