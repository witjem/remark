package boot

import (
	"log"
	"os"
	"time"

	"github.com/jinzhu/configor"
	"github.com/pkg/errors"
)

// Config maps yaml/json/toml config
type Config struct {
	SecretKey string `yaml:"secret" env:"SECRET" required:"true"`
	RemarkURL string `yaml:"url" env:"REMARK_URL" required:"true"`

	Sites       []string `yaml:"sites"`
	Port        int      `yaml:"port" default:"8080"`
	WebRoot     string   `yaml:"web" default:"./web"`
	ReadOnlyAge int      `yaml:"read_only_age" default:"0"`
	ImageProxy  bool     `yaml:"image_proxy" default:"false"`
	DevPasswd   string   `yaml:"dev_passwd" default:""`

	Storage struct {
		Type     string `yaml:"type" default:"bolt"`
		BoltPath string `yaml:"bolt_path" default:"./var"`
	} `yaml:"storage"`

	Admin struct {
		Email string   `yaml:"email"`
		IDs   []string `yaml:"ids"`
	} `yaml:"admins"`

	Limits struct {
		Body          int           `yaml:"max_body_size" default:"65536"`
		SocketTimeout time.Duration `yaml:"socket_timeout" default:"30s"`
		CommentSize   int           `yaml:"comment_size" default:"2048"`
		EditDuration  time.Duration `yaml:"edit_duration" default:"5m"`
	} `yaml:"limits"`

	Backup struct {
		Location string        `yaml:"location" default:"./var/backup"`
		MaxFiles int           `yaml:"max_files" default:"10"`
		Duration time.Duration `yaml:"duration" default:"24h"`
	} `yaml:"backup"`

	Auth struct {
		JwtExp    time.Duration `yaml:"jwt_exp" default:"168h"`
		Providers []struct {
			Name string `yaml:"name"`
			CID  string `yaml:"cid"`
			CSEC string `yaml:"csec"`
		} `yaml:"providers" required:"true"`
	}

	Avatar struct {
		Type   string `yaml:"type" default:"fs"`
		FsPath string `yaml:"fs_path" default:"./var/avatars"`
	} `yaml:"avatar"`

	Cache struct {
		Items int   `yaml:"items" default:"1000"`
		Value int   `yaml:"value_max_size" default:"100000"`
		Size  int64 `yaml:"cache_size" default:"50000000"`
	} `yaml:"cache"`

	Scores struct {
		Low      int `yaml:"low" default:"-10"`
		Critical int `yaml:"critical" default:"-20"`
	} `yaml:"scores"`
}

// NewConfig make new configuration from yaml file
func NewConfig(file string) (*Config, error) {
	var conf Config
	if err := configor.New(&configor.Config{Debug: false, ErrorOnUnmatchedKeys: true}).Load(&conf, file); err != nil {
		return nil, errors.Wrapf(err, "failed to load %s", file)
	}
	if err := os.Setenv("SECRET", "removed"); err != nil {
		log.Printf("[WARN] can't clear SECRET env, %s", err)
	}
	return &conf, nil
}
