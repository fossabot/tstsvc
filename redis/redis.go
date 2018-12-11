package tstredis

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/ory/dockertest"
	dc "github.com/ory/dockertest/docker"

	"github.com/huangjunwen/tstsvc"
)

var (
	// Docker repository.
	Repository = "redis"

	// Default tag.
	DefaultTag = "3.2.12-alpine"

	// Default container expire time.
	DefaultExpire uint = 120
)

var (
	// Default options.
	defaultOptions = &Options{}
)

// Resource represents a test redis server.
type Resource struct {
	// Redis server docker container.
	*dockertest.Resource

	// Actual options.
	Options
}

// Options is options to run a redis test server.
type Options struct {
	// Tag of the repository. Default: DefaultTag.
	Tag string

	// If specified, data will be stored in this host directory.
	HostDataPath string

	// If specified, the port 6379/tcp will be mapped to it. Default: random port.
	HostPort uint16

	// Expire time (in seconds) of the container. Default: DefaultExpire.
	Expire uint
}

// Run is equivalent to RunFromPool(nil, opts).
func Run(opts *Options) (*Resource, error) {
	return RunFromPool(nil, opts)
}

// RunFromPool runs a test redis server. If pool is nil, tstsvc.DefaultPool() will be used.
// If opts is nil, the default options will be used.
func RunFromPool(pool *dockertest.Pool, opts *Options) (*Resource, error) {
	// Handle nil case.
	if pool == nil {
		pool = tstsvc.DefaultPool()
	}
	if opts == nil {
		opts = defaultOptions
	}

	// Collect options.
	res := &Resource{
		Options: *opts,
	}
	opts = &res.Options

	if opts.Tag == "" {
		opts.Tag = DefaultTag
	}
	if opts.HostPort == 0 {
		opts.HostPort = tstsvc.FreePort()
	}
	if opts.Expire == 0 {
		opts.Expire = DefaultExpire
	}

	// Run the container.
	runOpts := &dockertest.RunOptions{
		Repository: Repository,
		Tag:        opts.Tag,
		PortBindings: map[dc.Port][]dc.PortBinding{
			"6379/tcp": []dc.PortBinding{
				dc.PortBinding{
					HostIP:   "localhost",
					HostPort: fmt.Sprintf("%d", opts.HostPort),
				},
			},
		},
	}
	if opts.HostDataPath != "" {
		runOpts.Mounts = append(runOpts.Mounts, fmt.Sprintf("%s:/data", opts.HostDataPath))
	}

	var err error
	res.Resource, err = pool.RunWithOptions(runOpts)
	if err != nil {
		return nil, err
	}

	// Set expire of the container.
	res.Resource.Expire(opts.Expire)

	// Wait.
	if err := pool.Retry(func() error {
		client := res.Client()
		defer client.Close()
		return client.Ping().Err()
	}); err != nil {
		res.Close()
		return nil, err
	}

	return res, nil
}

// Addr returns the addr to connect to the test server.
func (res *Resource) Addr() string {
	return fmt.Sprintf("localhost:%d", res.Options.HostPort)
}

// Client returns a redis client to the test server.
func (res *Resource) Client() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: res.Addr(),
	})
}