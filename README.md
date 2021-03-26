**forked from https://github.com/sourcegraph/checkup**

**Checkup is distributed, lock-free, self-hosted health checks and status pages, written in Go.**

**It features an elegant, minimalistic CLI and an idiomatic Go library. They are completely interoperable and their configuration is beautifully symmetric.**

This tool is a work-in-progress. Please use liberally (with discretion) and report any bugs!



## Intro

Checkup can be customized to check up on any of your sites or services at any time, from any infrastructure, using any storage provider of your choice (assuming an integration exists for your storage provider). The status page can be customized to your liking since you can do your checks however you want. The status page is also mobile-responsive.

Checkup currently supports these checkers:

- HTTP
- TCP (+TLS)
- DNS
- TLS
- Backup S3

Checkup implements these storage providers:

- Local file system
- GitHub
- SQL (sqlite3 or PostgreSQL)

Checkup can even send notifications through your service of choice (if an integration exists).


## How it Works

There are 3 components:

1. **Storage.** You set up storage space for the results of the checks.

2. **Checks.** You run checks on whatever endpoints you have as often as you want.

3. **Status Page.** You host the status page. The status page downloads recent check files from storage and renders the results client-side.


## Quick Start

[Download Checkup](https://github.com/Sparklane/checkup/releases/latest) for your platform and put it in your PATH, or install from source:

```bash
$ go get -u github.com/Sparklane/checkup/cmd/checkup
```

You'll need Go 1.8 or newer. Verify it's installed properly:

```bash
$ checkup --help
```

Then follow these instructions to get started quickly with Checkup.


### Create your Checkup config

You can configure Checkup entirely with a simple JSON document. You should configure storage and at least one checker. Here's the basic outline:

```json
{
	"checkers": [
		// checker configurations go here
	],

	"storage": {
		// storage configuration goes here
	},

	"notifier": {
		// notifier configuration goes here
	}
}
```

Save the checkup configuration file as `checkup.json` in your working directory.

We will show JSON samples below, to get you started. **But please [refer to the godoc](https://godoc.org/github.com/Sparklane/checkup) for a comprehensive description of each type of checker, storage, and notifier you can configure!**

Here are the configuration structures you can use, which are explained fully [in the godoc](https://godoc.org/github.com/Sparklane/checkup). **Only the required fields are shown, so consult the godoc for more.**

#### HTTP Checkers

**[godoc: HTTPChecker](https://godoc.org/github.com/Sparklane/checkup#HTTPChecker)**

```json
{
	"type": "http",
	"endpoint_name": "Example HTTP",
	"endpoint_url": "http://www.example.com"
	// for more fields, see the godoc
}
```


#### TCP Checkers

**[godoc: TCPChecker](https://godoc.org/github.com/Sparklane/checkup#TCPChecker)**

```json
{
	"type": "tcp",
	"endpoint_name": "Example TCP",
	"endpoint_url": "example.com:80"
}
```

#### DNS Checkers

**[godoc: DNSChecker](https://godoc.org/github.com/Sparklane/checkup#DNSChecker)**

```json
{
	"type": "dns",
	"endpoint_name": "Example of endpoint_url looking up host.example.com",
	"endpoint_url": "ns.example.com:53",
	"hostname_fqdn": "host.example.com"
}
```

#### TLS Checkers

**[godoc: TLSChecker](https://godoc.org/github.com/Sparklane/checkup#TLSChecker)**

```json
{
	"type": "tls",
	"endpoint_name": "Example TLS Protocol Check",
	"endpoint_url": "www.example.com:443"
}
```

#### Backup S3

```json
{
	"type": "backup:s3",
	"endpoint_name": "backup-s3",
	"region": "eu-west-1",
	"bucket_name": "BUCKET NAME",
	"min_age_threshold": "36h",
	"min_size_threshold": 1048576
}
```

#### Backup AMI

```json
{
	"type": "backup:ami",
	"endpoint_name": "backup-ami",
	"region": "eu-west-1",
	"ami_prefix": "AMI_PREFIX",
	"min_age_threshold": "36h"
}
```

#### Backup RDS

```json
{
		"type": "backup:rds",
		"endpoint_name": "backup-rds",
		"region": "eu-west-1",
		"instance": "RDS_INSTANCE",
		"min_age_threshold": "36h"
}
```

#### File System Storage

**[godoc: FS](https://godoc.org/github.com/Sparklane/checkup#FS)**

```json
{
	"provider": "fs",
	"dir": "/path/to/your/check_files",
	"url": "http://127.0.0.1:2015/check_files"
}
```

Then fill out [config.js](https://github.com/Sparklane/checkup/blob/master/statuspage/js/config.js) so the status page knows how to load your check files.

#### GitHub Storage

**[godoc: GitHub](https://godoc.org/github.com/Sparklane/checkup#GitHub)**

```json
{
	"provider": "github",
	"access_token": "some_api_access_token_with_repo_scope",
	"repository_owner": "owner",
	"repository_name": "repo",
	"committer_name": "Commiter Name",
	"committer_email": "you@yours.com",
	"branch": "gh-pages",
	"dir": "updates"
}
```

Where "dir" is a subdirectory within the repo to push all the check files. Setup instructions:

1. Create a repository.
2. Copy the contents of `statuspage/` from this repo to the root of your new repo.
3. Create `updates/.gitkeep`.
4. Enable GitHub Pages in your settings for your desired branch.

#### SQL Storage (sqlite3/PostgreSQL)

**[godoc: SQL](https://godoc.org/github.com/Sparklane/checkup#SQL)**

Postgres or sqlite3 databases can be used as storage backends.

sqlite database file configuration:
```json
{
	"provider": "sql",
	"sqlite_db_file": "/path/to/your/sqlite.db"
}
```

postgresql database file configuration:
```json
{
	"provider": "sql",
	"postgresql": {
		"user": "postgres",
		"dbname": "dbname",
		"host": "localhost",
		"port": 5432,
		"password": "password",
		"sslmode": "disable"
	}
}
```

The SQL engine used depends on which one is configured.

For all database backends, the database must exist and a "checks" table should be created:

```
CREATE TABLE checks (name TEXT NOT NULL PRIMARY KEY, timestamp INT8, results TEXT);
```

Currently the status page does not support SQL storage.

#### Slack notifier

Enable notifications in Slack with this Notifier configuration:
```json
{
	"name": "slack",
	"username": "username",
	"channel": "#channel-name",
	"webhook": "webhook-url"
}
```

Follow these instructions to [create a webhook](https://get.slack.help/hc/en-us/articles/115005265063-Incoming-WebHooks-for-Slack).


## Setting up the status page

In statuspage/js, use the contents of [config.js](https://github.com/Sparklane/checkup/blob/master/statuspage/js/config.js), which is used by the status page. This is where you specify how to access the storage system you just provisioned for check files.

As you perform checks, the status page will update every so often with the latest results. **Only checks that are stored will appear on the status page.**


## Performing checks

You can run checks many different ways: cron, AWS Lambda, or a time.Ticker in your own Go program, to name a few. Checks should be run on a regular basis. How often you run checks depends on your requirements and how much time you render on the status page.

For example, if you run checks every 10 minutes, showing the last 24 hours on the status page will require 144 check files to be downloaded on each page load. You can distribute your checks to help avoid localized network problems, but this multiplies the number of files by the number of nodes you run checks on, so keep that in mind.

Performing checks with the `checkup` command is very easy.

Just `cd` to the folder with your `checkup.json` from earlier, and checkup will automatically use it:

```bash
$ checkup
```

The vanilla checkup command runs a single check and prints the results to your screen, but does not save them to storage for your status page.

To store the results instead, use `--store`:

```bash
$ checkup --store
```

If you want Checkup to loop forever and perform checks and store them on a regular interval, use this:

```bash
$ checkup every 10m
```

And replace the duration with your own preference. In addition to the regular `time.ParseDuration()` formats, you can use shortcuts like `second`, `minute`, `hour`, `day`, or `week`.

You can also get some help using the `-h` option for any command or subcommand.


## Posting status messages

Site reliability engineers should post messages when there are incidents or other news relevant for a status page. This is also very easy:

```bash
$ checkup message --about=Example "Oops. We're trying to fix the problem. Stay tuned."
```

This stores a check file with your message attached to the result for a check named "Example" which you configured in `checkup.json` earlier.


## Doing all that, but with Go

Checkup is as easy to use in a Go program as it is on the command line.


### Using Go to perform checks

First, `go get github.com/Sparklane/checkup` and import it. Then configure it:

```go
c := checkup.Checkup{
	Checkers: []checkup.Checker{
		checkup.HTTPChecker{Name: "Example (HTTP)", URL: "http://www.example.com", Attempts: 5},
		checkup.HTTPChecker{Name: "Example (HTTPS)", URL: "https://www.example.com", Attempts: 5},
		checkup.TCPChecker{Name:  "Example (TCP)", URL:  "www.example.com:80", Attempts: 5},
		checkup.TCPChecker{Name:  "Example (TCP SSL)", URL:  "www.example.com:443", Attempts: 5, TLSEnabled: true},
		checkup.TCPChecker{Name:  "Example (TCP SSL, self-signed certificate)", URL:  "www.example.com:443", Attempts: 5, TLSEnabled: true, TLSCAFile: "testdata/ca.pem"},
		checkup.TCPChecker{Name:  "Example (TCP SSL, validation disabled)", URL:  "www.example.com:8443", Attempts: 5, TLSEnabled: true, TLSSkipVerify: true},
		checkup.DNSChecker{Name:  "Example DNS test of ns.example.com:53 looking up host.example.com", URL:  "ns.example.com:53", Host: "host.example.com", Attempts: 5},
	},
}
```

This sample checks 2 endpoints (HTTP and HTTPS). Each check consists of 5 attempts so as to smooth out the final results a bit. Notice the `CheckExpiry` value.

Then, to run checks every 10 minutes:

```go
c.CheckAndStoreEvery(10 * time.Minute)
select {}
```

`CheckAndStoreEvery()` returns a `time.Ticker` that you can stop, but in this case we just want it to run forever, so we block forever using an empty `select`.


### Using Go to post status messages

Simply perform a check, add the message to the corresponding result, and then store it:

```go
results, err := c.Check()
if err != nil {
	// handle err
}

results[0].Message = "We're investigating connectivity issues."

err = c.Storage.Store(results)
if err != nil {
	// handle err
}
```

Of course, real status messages should be as descriptive as possible. You can use HTML in them.


## Building

```bash
make
```

This will create a checkup binary in the root project folder.
