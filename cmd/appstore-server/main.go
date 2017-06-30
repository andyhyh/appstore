package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/UNINETT/appstore/cmd/appstore-server/api"
	"github.com/UNINETT/appstore/cmd/appstore-server/dashboard"
	"github.com/UNINETT/appstore/pkg/helmutil"
	"github.com/UNINETT/appstore/pkg/logger"
	"github.com/UNINETT/appstore/pkg/templateutil"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	log "github.com/Sirupsen/logrus"

	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

const (
	stableRepository         = "stable"
	localRepository          = "local"
	localRepositoryIndexFile = "index.yaml"
)

var startTime time.Time
var (
	stableRepositoryURL = "https://kubernetes-charts.storage.googleapis.com"
	// This is the IPv4 loopback, not localhost, because we have to force IPv4
	// for Dockerized Helm: https://github.com/kubernetes/helm/issues/1410
	localRepositoryURL = "http://127.0.0.1:8879/charts"
)

const version string = "v1"

func main() {
	debug := flag.Bool("debug", false, "Enable debug output")
	port := flag.Int("port", 8080, "The port to use when hosting the server")
	tillerHost := flag.String("host", os.Getenv(helm_env.HostEnvVar), "Address of tiller. Defaults to $HELM_HOST")
	flag.Parse()

	settings := helmutil.InitHelmSettings(*debug, *tillerHost)

	if err := ensureDirectories(settings.Home); err != nil {
		panic(err)
	}
	if err := ensureDefaultRepos(settings.Home, settings, false); err != nil {
		panic(err)
	}
	if err := ensureRepoFileFormat(settings.Home.RepositoryFile()); err != nil {
		panic(err)
	}

	baseRouter := chi.NewRouter()

	baseRouter.Use(middleware.RequestID)
	baseRouter.Use(middleware.RealIP)
	baseRouter.Use(logger.RequestLogger)
	baseRouter.Use(middleware.Recoverer)
	baseRouter.Use(middleware.CloseNotify)
	baseRouter.Use(middleware.Timeout(60 * time.Second))

	baseRouter.Mount("/api", api.CreateAPIRouter(settings))
	templates, err := templateutil.ProcessTemplates("ui/templates/")
	if err != nil {
		log.Fatal(err)
	}
	baseRouter.Mount("/", dashboard.CreateDashboardRouter(settings, templates))
	baseRouter.Get("/healthz", healthzHandler)

	FileServer(baseRouter, "/static", http.Dir("ui/static"))

	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)
	log.Debug("Starting server on port ", *port)
	log.Debug("Tiller host: ", settings.TillerHost)
	startTime = time.Now()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), baseRouter))
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}

func ensureDirectories(home helmpath.Home) error {
	configDirectories := []string{
		home.String(),
		home.Repository(),
		home.Cache(),
		home.LocalRepository(),
		home.Plugins(),
		home.Starters(),
		home.Archive(),
	}
	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			fmt.Printf("Creating %s \n", p)
			if err := os.MkdirAll(p, 0755); err != nil {
				return fmt.Errorf("Could not create %s: %s", p, err)
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("%s must be a directory", p)
		}
	}

	return nil
}

func ensureDefaultRepos(home helmpath.Home, settings *helm_env.EnvSettings, skipRefresh bool) error {
	repoFile := home.RepositoryFile()
	if fi, err := os.Stat(repoFile); err != nil {
		fmt.Printf("Creating %s \n", repoFile)
		f := repo.NewRepoFile()
		sr, err := initStableRepo(home.CacheIndex(stableRepository), settings, skipRefresh)
		if err != nil {
			return err
		}
		lr, err := initLocalRepo(home.LocalRepository(localRepositoryIndexFile), home.CacheIndex("local"))
		if err != nil {
			return err
		}
		f.Add(sr)
		f.Add(lr)
		if err := f.WriteFile(repoFile, 0644); err != nil {
			return err
		}
	} else if fi.IsDir() {
		return fmt.Errorf("%s must be a file, not a directory", repoFile)
	}
	return nil
}

func initStableRepo(cacheFile string, settings *helm_env.EnvSettings, skipRefresh bool) (*repo.Entry, error) {
	c := repo.Entry{
		Name:  stableRepository,
		URL:   stableRepositoryURL,
		Cache: cacheFile,
	}
	r, err := repo.NewChartRepository(&c, getter.All(*settings))
	if err != nil {
		return nil, err
	}

	if skipRefresh {
		return &c, nil
	}

	// In this case, the cacheFile is always absolute. So passing empty string
	// is safe.
	if err := r.DownloadIndexFile(""); err != nil {
		return nil, fmt.Errorf("Looks like %q is not a valid chart repository or cannot be reached: %s", stableRepositoryURL, err.Error())
	}

	return &c, nil
}

func initLocalRepo(indexFile, cacheFile string) (*repo.Entry, error) {
	if fi, err := os.Stat(indexFile); err != nil {
		i := repo.NewIndexFile()
		if err := i.WriteFile(indexFile, 0644); err != nil {
			return nil, err
		}

		//TODO: take this out and replace with helm update functionality
		os.Symlink(indexFile, cacheFile)
	} else if fi.IsDir() {
		return nil, fmt.Errorf("%s must be a file, not a directory", indexFile)
	}

	return &repo.Entry{
		Name:  localRepository,
		URL:   localRepositoryURL,
		Cache: cacheFile,
	}, nil
}

func ensureRepoFileFormat(file string) error {
	r, err := repo.LoadRepositoriesFile(file)
	if err == repo.ErrRepoOutOfDate {
		fmt.Printf("Updating repository file format...")
		if err := r.WriteFile(file, 0644); err != nil {
			return err
		}
	}

	return nil
}
