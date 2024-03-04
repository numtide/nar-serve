# nar-serve - serve NAR file content

Push your build artifacts to one place.

All the files in https://cache.nixos.org are packed in NAR files which makes
them not directly accessible. This service allows to download, decompress,
unpack and serve any file in the cache on the fly.

## Use cases

* Avoid publishing build artifacts to both the binary cache and another service.
* Allows to share build results easily.
* Inspect the content of a NAR file.

## Development

Inside the provided nix shell run:

```shell
./start-dev
```

This will create a small local server with live reload that emulates now.sh.

Currently, the default port is 8383. You can change it by setting the `PORT` environment variable.

## Usage

Store contents can be fetched via a simple HTTP GET request.

Append any store path to the hostname to fetch and unpack it on
the fly. That's it.

E.g.:

* https://nar-serve.zimbatm.now.sh/nix/store/barxv95b8arrlh97s6axj8k7ljn7aky1-go-1.12/share/go/doc/effective_go.html

NAR archives also contain information about the executable bit for each contained file.
nar-serve uses a custom HTTP header named `NAR-executable` to indicate whether the fetched file would be executable.

## Configuration

You can use the following environment variables to configure nar-serve:

| Name | Default value | Description |
|:--   |:--            |:-- |
| `PORT` | `8383` | Port number on which nar-service listens |
| `NAR_CACHE_URL` | `https://cache.nixos.org` | The URL of the Nix store from which NARs are fetched |

## Contributing

Contributions are welcome!

Before adding any new feature it might be best to first discuss them by
creating a new issue in https://github.com/numtide/nar-serve/issues .

All code is licensed under the Apache 2.0 license.
