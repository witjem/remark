package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/bbolt"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/jinzhu/configor"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest/api"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/cache"
	"github.com/umputun/remark/app/rest/proxy"
	"github.com/umputun/remark/app/store/engine"
	"github.com/umputun/remark/app/store/service"
)

// Opts with command line flags and env
// nolint:maligned
type Opts struct {
	Config string `short:"f" long:"config" env:"CONFIG" default:"config.yml" description:"config file"`
	Dbg    bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}

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
		Body          int           `yaml:"body" default:"65536"`
		SocketTimeout time.Duration `yaml:"socket" default:"30s"`
		CommentSize   int           `yaml:"comment_size" default:"2048"`
		EditDuration  time.Duration `yaml:"edit_duration" default:"5m"`
	} `yaml:"limits"`

	Backup struct {
		Location string        `yaml:"backup" default:"./var/backup"`
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
		Value int   `yaml:"value" default:"100000"`
		Size  int64 `yaml:"size" default:"50000000"`
	} `yaml:"cache"`

	Scores struct {
		Low      int `yaml:"low" default:"-10"`
		Critical int `yaml:"critical" default:"-20"`
	} `yaml:"scores"`
}

var revision = "unknown"

// Application holds all active objects
type Application struct {
	Config
	debug       bool
	restSrv     *api.Rest
	migratorSrv *api.Migrator
	exporter    migrator.Exporter
	terminated  chan struct{}
}

func main() {
	fmt.Printf("remark42 %s\n", revision)

	var opts Opts
	p := flags.NewParser(&opts, flags.Default)
	if _, e := p.ParseArgs(os.Args[1:]); e != nil {
		os.Exit(1)
	}
	log.Print("[INFO] started remark")

	var conf Config
	if e := configor.New(&configor.Config{Debug: false, ErrorOnUnmatchedKeys: true}).Load(&conf, opts.Config); e != nil {
		log.Fatalf("failed to load %s, %s", opts.Config, e)
	}
	if err := os.Setenv("SECRET", "removed"); err != nil {
		log.Printf("[WARN] can't clear SECRET env, %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch signal and invoke graceful termination
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Print("[WARN] interrupt signal")
		cancel()
	}()

	app, err := New(conf, opts.Dbg)
	if err != nil {
		log.Fatalf("[ERROR] failed to setup application, %+v", err)
	}
	app.Run(ctx)
	log.Printf("[INFO] remark terminated")
}

// New prepares application and return it with all active parts
// doesn't start anything
func New(conf Config, dbg bool) (*Application, error) {
	setupLog(dbg)

	if len(conf.Sites) == 0 {
		conf.Sites = append(conf.Sites, "remark")
	}
	if err := makeDirs(conf.Storage.BoltPath, conf.Backup.Location, conf.Avatar.FsPath); err != nil {
		return nil, err
	}

	if !strings.HasPrefix(conf.RemarkURL, "http://") && !strings.HasPrefix(conf.RemarkURL, "https://") {
		return nil, errors.Errorf("invalid remark42 url %s", conf.RemarkURL)
	}

	boltStore, err := makeBoltStore(conf.Sites, conf.Storage.BoltPath)
	if err != nil {
		return nil, err
	}
	dataService := service.DataStore{
		Interface:      boltStore,
		EditDuration:   5 * time.Minute,
		Secret:         conf.SecretKey,
		MaxCommentSize: conf.Limits.CommentSize,
	}

	loadingCache, err := cache.NewLoadingCache(cache.MaxValSize(conf.Cache.Value), cache.MaxKeys(conf.Cache.Items),
		cache.MaxCacheSize(conf.Cache.Size), cache.PostFlushFn(postFlushFn(conf.Sites, conf.Port)))
	if err != nil {
		return nil, err
	}

	jwtService := auth.NewJWT(conf.SecretKey, strings.HasPrefix(conf.RemarkURL, "https://"), conf.Auth.JwtExp)

	avatarProxy := &proxy.Avatar{
		Store:     proxy.NewFSAvatarStore(conf.Avatar.FsPath),
		RoutePath: "/api/v1/avatar",
		RemarkURL: strings.TrimSuffix(conf.RemarkURL, "/"),
	}

	exporter := &migrator.Remark{DataStore: &dataService}

	migr := &api.Migrator{
		Version:        revision,
		Cache:          loadingCache,
		NativeImporter: &migrator.Remark{DataStore: &dataService},
		DisqusImporter: &migrator.Disqus{DataStore: &dataService},
		NativeExported: &migrator.Remark{DataStore: &dataService},
		SecretKey:      conf.SecretKey,
	}

	srv := &api.Rest{
		Version:     revision,
		DataService: dataService,
		Exporter:    exporter,
		WebRoot:     conf.WebRoot,
		RemarkURL:   conf.RemarkURL,
		ImageProxy:  &proxy.Image{Enabled: conf.ImageProxy, RoutePath: "/api/v1/img", RemarkURL: conf.RemarkURL},
		AvatarProxy: avatarProxy,
		ReadOnlyAge: conf.ReadOnlyAge,
		Authenticator: auth.Authenticator{
			JWTService: jwtService,
			Admins:     conf.Admin.IDs,
			AdminEmail: conf.Admin.Email,
			Providers:  makeAuthProviders(jwtService, avatarProxy, dataService, conf),
			DevPasswd:  conf.DevPasswd,
		},
		Cache: loadingCache,
	}

	// no admin email, use admin@domain
	if srv.Authenticator.AdminEmail == "" {
		if u, err := url.Parse(conf.RemarkURL); err == nil {
			srv.Authenticator.AdminEmail = "admin@" + u.Host
		}
	}

	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = conf.Scores.Low, conf.Scores.Critical
	tch := make(chan struct{})
	return &Application{Config: conf, restSrv: srv, migratorSrv: migr, exporter: exporter, debug: dbg, terminated: tch}, nil
}

// Run all application objects
func (a *Application) Run(ctx context.Context) {
	if a.DevPasswd != "" {
		log.Printf("[WARN] running in dev mode")
	}

	go func() {
		// shutdown on context cancellation
		<-ctx.Done()
		a.restSrv.Shutdown()
		a.migratorSrv.Shutdown()
	}()
	a.activateBackup(ctx) // runs in goroutine for each site
	go a.migratorSrv.Run(a.Port + 1)
	a.restSrv.Run(a.Port)
	close(a.terminated)
}

// Wait for application completion (termination)
func (a *Application) Wait() {
	<-a.terminated
}

// activateBackup runs background backups for each site
func (a *Application) activateBackup(ctx context.Context) {
	for _, siteID := range a.Sites {
		backup := migrator.AutoBackup{
			Exporter:       a.exporter,
			BackupLocation: a.Backup.Location,
			SiteID:         siteID,
			KeepMax:        a.Backup.MaxFiles,
			Duration:       a.Backup.Duration,
		}
		go backup.Do(ctx)
	}
}

// makeBoltStore creates store for all sites
func makeBoltStore(siteNames []string, path string) (engine.Interface, error) {
	sites := []engine.BoltSite{}
	for _, site := range siteNames {
		sites = append(sites, engine.BoltSite{SiteID: site, FileName: fmt.Sprintf("%s/%s.db", path, site)})
	}
	result, err := engine.NewBoltDB(bolt.Options{Timeout: 30 * time.Second}, sites...)
	if err != nil {
		return nil, errors.Wrap(err, "can't initialize data store")
	}
	return result, nil
}

// mkdir -p for all dirs
func makeDirs(dirs ...string) error {

	// exists returns whether the given file or directory exists or not
	exists := func(path string) (bool, error) {
		_, err := os.Stat(path)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
		return true, err
	}

	for _, dir := range dirs {
		ex, err := exists(dir)
		if err != nil {
			return errors.Wrapf(err, "can't check directory status for %s", dir)
		}
		if !ex {
			if e := os.MkdirAll(dir, 0700); e != nil {
				return errors.Wrapf(err, "can't make directory %s", dir)
			}
		}
	}
	return nil
}

func makeAuthProviders(jwtService *auth.JWT, avatarProxy *proxy.Avatar, ds service.DataStore, conf Config) []auth.Provider {

	makeParams := func(cid, secret string) auth.Params {
		return auth.Params{
			JwtService:   jwtService,
			AvatarProxy:  avatarProxy,
			RemarkURL:    conf.RemarkURL,
			Cid:          cid,
			Csecret:      secret,
			Admins:       conf.Admin.IDs,
			SecretKey:    conf.SecretKey,
			IsVerifiedFn: ds.IsVerifiedFn(),
		}
	}

	providers := []auth.Provider{}

	for _, p := range conf.Auth.Providers {
		switch strings.ToLower(p.Name) {
		case "google":
			providers = append(providers, auth.NewGoogle(makeParams(p.CID, p.CID)))
		case "github":
			providers = append(providers, auth.NewGithub(makeParams(p.CID, p.CID)))
		case "facebook":
			providers = append(providers, auth.NewFacebook(makeParams(p.CID, p.CID)))
		case "yandex":
			providers = append(providers, auth.NewYandex(makeParams(p.CID, p.CID)))
		default:
			log.Printf("[WARN] unrecognized auth provider %s", p.Name)
		}
	}

	if len(providers) == 0 {
		log.Printf("[WARN] no auth providers defined")
	}
	return providers
}

// post-flush callback invoked by cache after each flush in async way
func postFlushFn(sites []string, port int) func() {

	return func() {
		// list of heavy urls for pre-heating on cache change
		urls := []string{
			"http://localhost:%d/api/v1/list?site=%s",
			"http://localhost:%d/api/v1/last/50?site=%s",
		}

		for _, site := range sites {
			for _, u := range urls {
				resp, err := http.Get(fmt.Sprintf(u, port, site))
				if err != nil {
					log.Printf("[WARN] failed to refresh cached list for %s, %s", site, err)
					return
				}
				if err = resp.Body.Close(); err != nil {
					log.Printf("[WARN] failed to close response body, %s", err)
				}
			}
		}
	}
}

func setupLog(dbg bool) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stdout,
	}

	log.SetFlags(log.Ldate | log.Ltime)

	if dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		filter.MinLevel = logutils.LogLevel("DEBUG")
	}
	log.SetOutput(filter)
}
