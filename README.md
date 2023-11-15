# seilfahrt

Tool to create a wiki page from a HedgeDoc

## Prerequisites

You must have pandoc installed and in your PATH variable for this to work!
Also you must have a access_token, access_secret, consumer_token and consumer_secret for your wiki.

## Usage

Create a config file with your access tokens and wiki settings and put it next to the executable.
Afterwards simply execute the executable with the -id command line flage and append the id of the Hedgedoc pad.

### Command Line Parameters

| Parameter | Description                                              | Default Value |
|-----------|----------------------------------------------------------|---------------|
| id        | The ID of the hedgedoc pad to be dowloaded and converted |               |
| c         | The path to the config file                              | ./config.toml |
| h         | If active prints all command line args                   |               |
