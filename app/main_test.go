package main

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	os.Args = []string{"test", "--dbg", "--config=/tmp/remark42-test.yml"}

	confFileName := "/tmp/remark42-test.yml"
	defer os.Remove(confFileName)
	ioutil.WriteFile(confFileName, []byte(testConfigFile), 0600)

	go func() {
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()
	st := time.Now()
	main()
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
}

var testConfigFile = `
secret: 123456
dev_passwd: password
url: https://demo.remark42.com
auth:
 providers:
  - name: google
    cid: 123456789
    csec: 09876543210987654321
`
