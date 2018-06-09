package boot

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	location := prepConf(t)
	defer os.Remove(location)
	_, err := NewConfig(location)
	assert.Nil(t, err)

	_, err = NewConfig("/dev/null")
	assert.NotNil(t, err)

}
func prepConf(t *testing.T) string {
	// prepare options
	configFile := `
secret: 123456
dev_passwd: password
url: https://demo.remark42.com
storage:
  type: bolt
  bolt_path: /tmp
backup:
  location: /tmp
avatar:
  fs_path: /tmp
auth:
  providers:
    - name: google
      cid: 123456789
      csec: 09876543210987654321
    - name: github
      cid: 123456789
      csec: 09876543210987654321
    - name: facebook
      cid: 123456789
      csec: 09876543210987654321
    - name: yandex
      cid: 123456789
      csec: 09876543210987654321
    - name: unknown
      cid: 123456789
      csec: 09876543210987654321
`
	confFileName := "/tmp/remark42-test.yml"
	os.Remove(confFileName)
	ioutil.WriteFile(confFileName, []byte(configFile), 0600)
	return confFileName
}
