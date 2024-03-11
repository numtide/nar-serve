# Tests

## With the minio package

```shell
mkdir nar
minio server ./nar
```

## With the pkgs.minio-client package

```shell
mc config host add mycloud http://127.0.0.1:9000 accesskey secretkey
mc mb mycloud/nar
AWS_ACCESS_KEY_ID=accesskey AWS_SECRET_ACCESS_KEY=secretkey nix copy --to "s3://nar?region=eu-west-1&endpoint=127.0.0.1:9000&scheme=http" /nix/store/irfa91bs2wfqyh2j9kl8m3rcg7h72w4m-curl-7.71.1-bin
```

## Run the test

```shell
go run main.go
```
