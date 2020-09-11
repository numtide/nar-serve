package integration_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/numtide/nar-serve/go-nix/libstore"
	"github.com/stretchr/testify/assert"
)

func cmd(env []string, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	return cmd
}

func TestHappyPath(t *testing.T) {
	assert := assert.New(t)
	accessKeyID := "Q3AM3UQ867SPQQA43P2F"
	secretAccessKey := "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"

	tempDir, err := ioutil.TempDir("", "nar-serve")
	homeDir := tempDir + "/home"
	configDir := tempDir + "/config"
	dataDir := tempDir + "/data"

	env := append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+accessKeyID,
		"AWS_SECRET_ACCESS_KEY="+secretAccessKey,
		"MINIO_ACCESS_KEY="+accessKeyID,
		"MINIO_SECRET_KEY="+secretAccessKey,
		"MINIO_REGION_NAME=us-east-1",
		"HOME="+homeDir,
	)

	if err != nil {
		t.Fatal("tmpdir error:", err)
	}
	defer os.RemoveAll(tempDir)

	// Start the server
	minios := cmd(env, "minio", "server", dataDir, "--config-dir", configDir)
	err = minios.Start()
	if err != nil {
		t.Fatal("minio error:", err)
	}
	defer func() {
		minios.Process.Kill()
		minios.Wait()
	}()

	minioc := cmd(env, "mc", "config", "host", "add", "narcloud", "http://127.0.0.1:9000", accessKeyID, secretAccessKey, "--config-dir", configDir, "--api", "s3v4")
	err = minioc.Run()
	if err != nil {
		t.Fatal("mc error:", err)
	}

	minio_bucket := cmd(env, "mc", "mb", "narcloud/nsbucket")
	err = minio_bucket.Run()
	if err != nil {
		t.Fatal("mc error:", err)
	}

	nix_copy := cmd(env, "nix", "copy", "--to", "s3://nsbucket?region=us-east-1&endpoint=127.0.0.1:9000&scheme=http", "/nix/store/irfa91bs2wfqyh2j9kl8m3rcg7h72w4m-curl-7.71.1-bin")
	err = nix_copy.Run()
	if err != nil {
		t.Fatal("nix-copy error:", err)
	}

	ctx := context.Background()

	tmpfile := filepath.Join(dataDir, "nsbucket/irfa91bs2wfqyh2j9kl8m3rcg7h72w4m.narinfo")
	_, err = os.Stat(tmpfile)

	if err != nil {
		if os.IsNotExist(err) {
			t.Fatal("File not exists")
		} else {
			t.Fatal("ERROR:", err)
		}
	}
	content, err := ioutil.ReadFile(tmpfile)

	if err != nil {
		t.Fatal(err)
	}

	// S3 binary cache storage
	r, err := libstore.NewBinaryCacheReader(ctx, "s3://nsbucket?region=us-east-1&endpoint=http://127.0.0.1:9000&scheme=http")
	if err != nil {
		t.Fatal("new binary cache error:", err)
	}

	os.Setenv("AWS_ACCESS_KEY_ID", accessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", secretAccessKey)
	obj, err := r.GetFile(ctx, "irfa91bs2wfqyh2j9kl8m3rcg7h72w4m.narinfo")
	if err != nil {
		t.Fatal("get file error:", err)
	}

	obj_content, err_read := ioutil.ReadAll(obj)
	if err_read != nil {
		t.Fatal(err_read)
	}

	same_content := bytes.Equal(content, obj_content)
	assert.True(same_content, "The content is not the same")

	is_exist, err := r.FileExists(ctx, "irfa91bs2wfqyh2j9kl8m3rcg7h72w4m.narinfo")
	if err != nil {
		t.Fatal("file exist error:", err)
	}
	assert.True(is_exist, "File is not existed")
	// Stop the server
	minios.Process.Kill()
}
