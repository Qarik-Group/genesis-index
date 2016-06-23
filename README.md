Genesis Index
=============

`genesis-index` is a small CF-ready application that tracks
stemcells and releases from [bosh.io](https://bosh.io) and other
places.  The [genesis][genesis] utility uses the index to look up
versions, URLs and SHA1 checksums of said releases and stemcells.


Using the CLI
=============

`indexer` is a small Bash script that provides a basic
command-line interface for dealing with the Genesis Index.

It obeys the following environment variables:

- `GENESIS_INDEX` - The base URL of the Genesis Index.  If not
  set, defaults to `https://genesis.starkandwayne.com`
- `GENESIS_CREDS` - The username and password for accessing the
  protected parts of the Index API, separated by a colon.
- `INDEXER_DEBUG` - Set to a non-empty value to enable debugging 

Here are the commands:

```
indexer version (release|stemcell) NAME [VERSION]
indexer show    (release|stemcell) NAME
indexer check   (release|stemcell) NAME VERSION
indexer create  (release|stemcell) NAME URL
indexer remove  (release|stemcell) NAME [VERSION]
indexer releases
indexer stemcells
indexer help
```

So, for example, to get the latest version of the SHIELD BOSH
release:

```
$ indexer version release shield
```

Or, to get the SHA1 sum of v19 of the Consul BOSH release:

```
$ indexer version release consul 19
```


API Overview
============

The Genesis Index API strives to be simple and clean

## Get a List of Tracked Releases

```
GET /v1/release
```

## Get All Release Versions

```
GET /v1/release/:name
```

## Get Release Metadata

```
GET /v1/release/:name/metadata
```

## Get The Latest Release Version

```
GET /v1/release/:name/latest
```

## Get a Specific Release Version

```
GET /v1/release/:name/v/:version
```

## Start Tracking a New Release

(this endpoint requires authentication)

```
POST /v1/release
{
  "name": "release name",
  "url":  "https://wherever/to/get/it?v={{version}}"
}
```

## Check a Specific Release Version

(this endpoint requires authentication)

```
PUT /v1/release/:name/v/:version
```

## Stop Tracking a Release

(this endpoint requires authentication)

```
DELETE /v1/release/:name
```

## Drop a Release Version

(this endpoint requires authentication)

```
DELETE /v1/release/:name/v/:version
```

## Get a List of Tracked Stemcells

```
GET /v1/stemcell
```

## Get All Stemcell Versions

```
GET /v1/stemcell/:name
```

## Get Stemcell Metadata

```
GET /v1/stemcell/:name/metadata
```

## Get The Latest Stemcell Version

```
GET /v1/stemcell/:name/latest
```

## Get a Specific Stemcell Version

```
GET /v1/stemcell/:name/v/:version
```

## Start Tracking a New Stemcell

(this endpoint requires authentication)

```
POST /v1/stemcell
{
  "name": "stemcell name",
  "url":  "https://wherever/to/get/it?v={{version}}"
}
```

## Check a Specific Stemcell Version

(this endpoint requires authentication)

```
PUT /v1/stemcell/:name/v/:version
```

## Stop Tracking a Stemcell

(this endpoint requires authentication)

```
DELETE /v1/stemcell/:name
```

## Drop a Stemcell Version

(this endpoint requires authentication)

```
DELETE /v1/stemcell/:name/v/:version
```


Installation And Operation
==========================

To deploy to Pivotal Web Services:

```
cf push
```

You need to bind a PostgreSQL database to your running app.  The
application will automatically detect the service if it is tagged
`postgres`.

The following environment variables should also be set:

- `AUTH_USERNAME` - The username for authenticated endpoints
- `AUTH_PASSWORD` - The password for authenticated endpoints


Pipelining The Updates
======================

Tracking all those versions and letting Genesis Index know when
they need updated is tedious work.  Let's make the robots do it!

The `pipeline/` directory contains the scripts for building a
concourse pipeline based off of the current configured set of
tracked releases and stemcells.  To use it:

```
./pipeline/repipe
```

At the moment, it's tied directly to the Stark & Wayne concourse
installation, under the alias `sw`.  That may change in the future

(Note: the `ci/` directory name is reserved for a future in which
we want / need to do CI/CD for the Genesis Index code / deployment
itself.)
if anyone is interested in more flexibility.
