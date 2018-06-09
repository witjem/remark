package boot

import (
	"log"
	"os"
	"time"

	"github.com/jinzhu/configor"
	"github.com/pkg/errors"
)

// Config maps yaml config
type Config struct {
	SecretKey string `yaml:"secret" env:"SECRET" required:"true"`
	RemarkURL string `yaml:"url" env:"REMARK_URL" required:"true"`

	Sites       []string `yaml:"sites"`
	Port        int      `yaml:"port" default:"8080"`
	WebRoot     string   `yaml:"web" default:"./web"`
	ReadOnlyAge int      `yaml:"read_only_age" default:"0"`
	ImageProxy  bool     `yaml:"image_proxy" default:"false"`
	DevPasswd   string   `yaml:"dev_passwd" default:""`

	Storage Store       `yaml:"storage"`
	Avatar  AvatarStore `yaml:"avatar"`
	Cache   Cache       `yaml:"cache"`
	Backup  Backup      `yaml:"backup"`

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

	Auth struct {
		JwtExp    time.Duration `yaml:"jwt_exp" default:"168h"`
		Providers []struct {
			Name string `yaml:"name"`
			CID  string `yaml:"cid"`
			CSEC string `yaml:"csec"`
		} `yaml:"providers" required:"true"`
	}

	Scores struct {
		Low      int `yaml:"low" default:"-10"`
		Critical int `yaml:"critical" default:"-20"`
	} `yaml:"scores"`
}

// Store is config for all supported stores
// supported types: bolt
type Store struct {
	Type string `yaml:"type" default:"bolt"`
	Bolt struct {
		Location string `yaml:"location" default:"./var"`
	} `yaml:"bolt"`
}

// AvatarStore is config for all supported avatar stores
// supported types: fs
type AvatarStore struct {
	Type string `yaml:"type" default:"fs"`
	FS   struct {
		Location string `yaml:"location" default:"./var/avatars"`
	} `yaml:"fs"`
}

// Cache is config for all supported caches
// supported types: mem
type Cache struct {
	Type  string `yaml:"type" default:"mem"`
	Items int    `yaml:"items" default:"1000"`
	Value int    `yaml:"value_max_size" default:"100000"`
	Size  int64  `yaml:"cache_size" default:"50000000"`
}

// Backup is config for all supported backups
// supported types: local
type Backup struct {
	Type  string `yaml:"type" default:"local"`
	Local struct {
		Location string `yaml:"location" default:"./var/backup"`
	} `yaml:"local"`
	MaxFiles int           `yaml:"max_files" default:"10"`
	Duration time.Duration `yaml:"duration" default:"24h"`
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
