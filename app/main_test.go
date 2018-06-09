package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jinzhu/configor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplication(t *testing.T) {
	app, ctx := prepApp(t, 18080, 500*time.Millisecond)
	go app.Run(ctx)
	time.Sleep(100 * time.Millisecond) // let server start

	// send ping
	resp, err := http.Get("http://localhost:18080/api/v1/ping")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	// add comment
	resp, err = http.Post("http://dev:password@localhost:18080/api/v1/comment", "json",
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	t.Log(string(body))

	assert.Equal(t, "admin@demo.remark42.com", app.restSrv.Authenticator.AdminEmail, "default admin email")

	app.Wait()
}

func TestApplicationFailed(t *testing.T) {
	conf, location := prepConf(t)
	defer os.Remove(location)

	// RO bolt location
	conf.Storage.BoltPath = "/dev/null"
	_, err := New(conf, true)
	assert.EqualError(t, err, "can't initialize data store: failed to make boltdb for /dev/null/remark.db: "+
		"open /dev/null/remark.db: not a directory")
	t.Log(err)

	// RO backup location
	conf.Storage.BoltPath = "/tmp"
	conf.Backup.Location = "/dev/null/not-writable"
	_, err = New(conf, true)
	assert.EqualError(t, err, "can't check directory status for /dev/null/not-writable: stat /dev/null/not-writable: not a directory")
	t.Log(err)

	// invalid url
	conf.Storage.BoltPath = "/tmp"
	conf.Backup.Location = "/tmp"
	conf.RemarkURL = "demo.remark42.com"
	_, err = New(conf, true)
	assert.EqualError(t, err, "invalid remark42 url demo.remark42.com")
	t.Log(err)
}

func TestApplicationShutdown(t *testing.T) {
	app, ctx := prepApp(t, 18090, 500*time.Millisecond)
	st := time.Now()
	app.Run(ctx)
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
	app.Wait()
}

func TestApplicationMainSignal(t *testing.T) {
	conf, location := prepConf(t)
	defer os.Remove(location)
	os.Remove(conf.Storage.BoltPath + "/remark.db")
	os.Args = []string{"--dbg", "--config=" + location}

	go func() {
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()
	st := time.Now()
	main()
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
}

func prepApp(t *testing.T, port int, duration time.Duration) (*Application, context.Context) {
	// prepare options

	conf, location := prepConf(t)
	defer os.Remove(location)
	os.Remove(conf.Storage.BoltPath + "/remark.db")

	conf.Port = port
	// create app
	app, err := New(conf, true)
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(duration)
		log.Print("[TEST] terminate app")
		cancel()
	}()
	return app, ctx
}

func prepConf(t *testing.T) (conf Config, location string) {
	// prepare options
	configFile := `
secret: 123456
dev_passwd: password
url: https://demo.remark42.com
storage:
  type: bolt
  bolt_path: /tmp
auth:
  providers:
    - name: google
      cid: 123456789
      csec: zxcvbnmasdfgh
    - name: github
      cid: 123456789
      csec: zxcvbnmasdfgh
    - name: facebook
      cid: 123456789
      csec: zxcvbnmasdfgh
    - name: yandex
      cid: 123456789
      csec: zxcvbnmasdfgh
`
	confFileName := "/tmp/remark42-test.yml"
	os.Remove(confFileName)
	ioutil.WriteFile(confFileName, []byte(configFile), 0600)

	err := configor.New(&configor.Config{Debug: false, ErrorOnUnmatchedKeys: true}).Load(&conf, confFileName)
	require.Nil(t, err)
	return conf, confFileName
}
