# gofollow
Follow Twitter users based on a search term

# Installation
This requires a working Go environment to run. Follow the steps [here](http://golang.org/doc/install) to install the Go environment.

Once Go is running, you can download and build the application using the following command:

<code>$ go get github.com/kompiuter/gofollow</code>

The executable can then be found under
``%GOPATH%\bin``.

# About
Running this command will go and follow users on Twitter which are related to the search term that you provide. It will prioritise following users based on user search (which tend to be more relevant) and once no more users are found through user search it will find users based on their tweets.

This is one of my first projects in Go so any feedback and/or PR's would be greatly appreciated.

# Usage
Get your keys from [Twitter](https://apps.twitter.com/), then set the following environment variables:
- TWITTER_CONSUMER_KEY
- TWITTER_CONSUMER_SECRET
- TWITTER_ACCESS_TOKEN
- TWITTER_ACCESS_TOKEN

How to set environment variables in [Windows systems](http://ss64.com/nt/set.html) and in [Unix systems](http://www.cyberciti.biz/faq/set-environment-variable-unix/).



To follow users related to Go:

<code>$ gofollow -s golang</code>

By default it will follow a maximum of 50 users. To change the maximum, use the ``-max`` flag:

<code>$ gofollow -s gopher -max 15</code>

A hard maximum of 100 exists so that you don't get limited by Twitter (too many API requests/too many follow requests).

## Query Operators

You may use any [query operator](https://dev.twitter.com/rest/public/search#query-operators) as defined by Twitter to refine your search.

### Useful Examples

Containing **both** ``golang`` and ``tutorial``:

<code>$ gofollow -s "golang tutorial"</code>

Containing **either** 'golang' or 'gopher' (or both):

<code>$ gofollow -s "golang OR gopher"</code>

Containing **exact phrase** 'open source':

<code>$ gofollow -s "\"open source\""</code>

*Character to escape double quotes may differ in your environment*




