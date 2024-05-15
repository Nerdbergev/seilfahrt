# seilfahrt

Tool to create a wiki page from a HedgeDoc

## Prerequisites

You must have pandoc installed and in your PATH variable for this to work!
Also you must have a access_token, access_secret, consumer_token and consumer_secret for your wiki.

## Usage

Create a config file with your access tokens and wiki settings and put it next to the executable.
Afterwards simply execute the executable with the -pad command line flag and append the url of the Hedgedoc pad.

Alternativley you can also run it with the -web option which launches a Webserver where you can enter the pads in an edit.

### Command Line Parameters

| Parameter | Description                            | Default Value |
|-----------|----------------------------------------|---------------|
| pad       | The URL of the hedgedoc pad.           |               |
| c         | The path to the config file            | ./config.toml |
| h         | If active prints all command line args |               |
| web       | Starts a webserver for interactive use | false         |
| port      | Defines the port the webserver runs on | 8080          |
