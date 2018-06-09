package boot

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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

	assert.Equal(t, 4, len(app.restSrv.Authenticator.Providers), "%+v", app.Auth.Providers)

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
	location := prepConf(t)
	var conf Config
	err := configor.New(&configor.Config{Debug: false, ErrorOnUnmatchedKeys: true}).Load(&conf, location)
	require.Nil(t, err)
	defer os.Remove(location)

	// RO bolt location
	conf.Storage.Bolt.Location = "/dev/null"
	_, err = NewApplication(&conf, "")
	assert.EqualError(t, err, "can't initialize data store: failed to make boltdb for /dev/null/remark.db: "+
		"open /dev/null/remark.db: not a directory")
	t.Log(err)

	// RO backup location
	conf.Storage.Bolt.Location = "/tmp"
	conf.Backup.Local.Location = "/dev/null/not-writable"
	_, err = NewApplication(&conf, "")
	assert.EqualError(t, err, "can't check directory status for /dev/null/not-writable: stat /dev/null/not-writable: not a directory")
	t.Log(err)

	// invalid url
	conf.Storage.Bolt.Location = "/tmp"
	conf.Backup.Local.Location = "/tmp"
	conf.RemarkURL = "demo.remark42.com"
	_, err = NewApplication(&conf, "")
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

func prepApp(t *testing.T, port int, duration time.Duration) (*Application, context.Context) {
	// prepare options

	location := prepConf(t)
	var conf Config
	err := configor.New(&configor.Config{Debug: false, ErrorOnUnmatchedKeys: true}).Load(&conf, location)
	require.Nil(t, err)
	defer os.Remove(location)
	os.Remove(conf.Storage.Bolt.Location + "/remark.db")

	conf.Port = port
	// create app
	app, err := NewApplication(&conf, "124")
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(duration)
		log.Print("[TEST] terminate app")
		cancel()
	}()
	return app, ctx
}