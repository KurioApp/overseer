// Enqueue
//
// The enqueue sub-command adds parsed tests to a central redis queue.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/cmaster11/overseer/parser"
	"github.com/cmaster11/overseer/test"
	"github.com/go-redis/redis"
	"github.com/google/subcommands"
)

type enqueueCmd struct {
	RedisDB          int
	RedisHost        string
	RedisPassword    string
	RedisSocket      string
	RedisDialTimeout time.Duration
	_r               *redis.Client
}

//
// Glue
//
func (*enqueueCmd) Name() string     { return "enqueue" }
func (*enqueueCmd) Synopsis() string { return "Enqueue a parsed configuration file" }
func (*enqueueCmd) Usage() string {
	return `enqueue :
  Add the tests from a parsed configuration file to a central redis queue.
`
}

//
// Flag setup.
//
func (p *enqueueCmd) SetFlags(f *flag.FlagSet) {

	//
	// Create the default options here
	//
	// This is done so we can load defaults via a configuration-file
	// if present.
	//
	var defaults enqueueCmd
	defaults.RedisHost = "localhost:6379"
	defaults.RedisPassword = ""
	defaults.RedisDB = 0
	defaults.RedisSocket = ""
	defaults.RedisDialTimeout = 5 * time.Second

	//
	// If we have a configuration file then load it
	//
	if len(os.Getenv("OVERSEER")) > 0 {
		cfg, err := ioutil.ReadFile(os.Getenv("OVERSEER"))
		if err == nil {
			err = json.Unmarshal(cfg, &defaults)
			if err != nil {
				fmt.Printf("WARNING: Error loading overseer.json - %s\n",
					err.Error())
			}
		} else {
			fmt.Printf("WARNING: Failed to read configuration-file - %s\n", err.Error())
		}
	}

	f.IntVar(&p.RedisDB, "redis-db", defaults.RedisDB, "Specify the database-number for redis.")
	f.StringVar(&p.RedisHost, "redis-host", defaults.RedisHost, "Specify the address of the redis queue.")
	f.StringVar(&p.RedisPassword, "redis-pass", defaults.RedisPassword, "Specify the password for the redis queue.")
	f.StringVar(&p.RedisSocket, "redis-socket", defaults.RedisSocket, "If set, will be used for the redis connections.")
	f.DurationVar(&p.RedisDialTimeout, "redis-timeout", defaults.RedisDialTimeout, "Redis connection timeout.")
}

//
// This is a callback invoked by the parser when a job
// has been successfully parsed.
//
func (p *enqueueCmd) enqueueTest(tst test.Test) error {
	_, err := p._r.RPush("overseer.jobs", tst.Input).Result()
	return err
}

//
// Entry-point.
//
func (p *enqueueCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	//
	// Connect to the redis-host.
	//
	if p.RedisSocket != "" {
		p._r = redis.NewClient(&redis.Options{
			Network:     "unix",
			Addr:        p.RedisSocket,
			Password:    p.RedisPassword,
			DB:          p.RedisDB,
			DialTimeout: p.RedisDialTimeout,
		})
	} else {
		p._r = redis.NewClient(&redis.Options{
			Addr:        p.RedisHost,
			Password:    p.RedisPassword,
			DB:          p.RedisDB,
			DialTimeout: p.RedisDialTimeout,
		})
	}

	//
	// And run a ping, just to make sure it worked.
	//
	_, err := p._r.Ping().Result()
	if err != nil {
		fmt.Printf("Redis connection failed: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	//
	// For each file on the command-line we can now parse and
	// enqueue the jobs
	//
	for _, file := range f.Args() {

		//
		// Create an object to parse our file.
		//
		helper := parser.New()

		//
		// For each parsed job call `enqueueTest`.
		//
		errParse := helper.ParseFile(file, p.enqueueTest)

		//
		// Did we see an error?
		//
		if errParse != nil {
			fmt.Printf("Error parsing file: %s\n", errParse)
			return subcommands.ExitFailure
		}

		// Did we read from stdin?
		if file == "-" {
			break
		}
	}

	return subcommands.ExitSuccess
}
