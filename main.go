package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/sync/errgroup"
)

type config struct {
	host    string
	port    string
	pass    string
	db      int
	mode    string
	workers int
	length  int
	max     int
	prefix  string
}

func run(ctx context.Context, config config) error {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.host, config.port),
		Password: config.pass,
		DB:       config.db,
	})

	val := generateString(config.length)

	g, ctx := errgroup.WithContext(ctx)

	for i := 1; i <= config.workers; i++ {
		id := i

		g.Go(func() error {
			mem, err := redisMemoryUsage(ctx, client)
			if err != nil {
				return err
			}
			if config.max != 0 && mem >= config.max*1000000 {
				return errors.New(fmt.Sprintf("Memory already above %d Mo, exiting", config.max))
			}
			for config.max == 0 || mem < config.max*1000000 {
				switch config.mode {
				case "get":
					key := "test"

					// insert random data
					fmt.Printf("worker %d: getting data...\n", id)
					err = client.Get(ctx, key).Err()
					if err != nil && err != redis.Nil { // ignoring when key does not exist
						return err
					}

				case "set":
					key := fmt.Sprintf("%s%s", config.prefix, uuid.NewV4().String())

					// insert random data
					fmt.Printf("worker %d: inserting data...\n", id)
					err = client.Set(ctx, key, val, 0).Err()
					if err != nil {
						return err
					}

				default:
					return errors.New(fmt.Sprintf("Unrecognized mode %s, exiting", config.mode))
				}

				// recalculate memory usage
				mem, err = redisMemoryUsage(ctx, client)
				if err != nil {
					return err
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	fmt.Printf("max memory %d Mo reached, exiting\n", config.max)
	return nil
}

func main() {
	var config config
	flag.StringVar(&config.host, "host", "localhost", "Redis host")
	flag.StringVar(&config.port, "port", "6379", "Redis port")
	flag.StringVar(&config.pass, "pass", "", "Redis password")
	flag.IntVar(&config.db, "db", 0, "Redis database number")
	flag.StringVar(&config.mode, "mode", "get", "Whether to run get or set commands")
	flag.IntVar(&config.workers, "workers", 100, "Number of parallel workers")
	flag.IntVar(&config.length, "length", 50000, "Length of generated values")
	flag.IntVar(&config.max, "max", 0, "Max memory in Mo (0 for unlimited)")
	flag.StringVar(&config.prefix, "prefix", "", "Prefix to append to keys")
	flag.Parse()

	ctx := context.Background()

	if err := run(ctx, config); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}

// generateString generate a string of length n.
func generateString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// redisMemoryUsage collects the used_memory from the command INFO MEMORY.
func redisMemoryUsage(ctx context.Context, client *redis.Client) (int, error) {
	res, err := client.Do(ctx, "INFO", "MEMORY").Result()
	if err != nil {
		return 0, err
	}

	memData, ok := res.(string)
	if !ok {
		return 0, errors.New("can’t gather memory")
	}

	re := regexp.MustCompile("used_memory:[0-9]+")
	line := re.FindString(memData)

	sliced := strings.Split(line, ":")
	if len(sliced) < 2 {
		return 0, errors.New("error retrieving memory")
	}

	mem, err := strconv.Atoi(sliced[1])
	if err != nil {
		return 0, err
	}

	return mem, nil
}
