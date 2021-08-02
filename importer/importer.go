package importer

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	redisClient "github.com/githubixx/pinbackup/redis"
)

// Config holds the configuration
type Config struct {
	ImportDir        string
	UserDatabaseName string
}

// Board holds the board informaton
type Board struct {
	Name  string
	Files []string
}

// User holds the username and boards of the user
type User struct {
	Username string
	Boards   map[string][]string
}

// StartImport imports all users and boards that exists in the specified
// directory.
func StartImport(ctx context.Context, config *Config) error {
	g, ctx := errgroup.WithContext(ctx)

	log.Trace().
		Str("method", "EnqueueBoard").
		Msg("Getting Redis connection")

	redisClient.InitPool(viper.GetString("redis-host"), viper.GetInt("redis-port"))
	conn, err := redisClient.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	users := make(chan User)

	g.Go(func() error {
		defer close(users)
		allUsers, err := getAllUser(config.ImportDir)
		if err != nil {
			return err
		}

		for _, user := range allUsers {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case users <- user:
			}
		}

		return nil
	})

	boards := make(chan User)
	nWorkers1 := 5
	workers1 := int32(nWorkers1)

	for i := 0; i < nWorkers1; i++ {
		g.Go(func() error {
			defer func() {
				if atomic.AddInt32(&workers1, -1) == 0 {
					close(boards)
				}
			}()

			for u := range users {
				if err := getBoards(config.ImportDir, &u); err != nil {
					return err
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case boards <- u:
				}
			}

			return nil
		})
	}

	files := make(chan User)
	nWorkers2 := 5
	workers2 := int32(nWorkers2)

	for i := 0; i < nWorkers2; i++ {
		g.Go(func() error {
			defer func() {
				if atomic.AddInt32(&workers2, -1) == 0 {
					close(files)
				}
			}()
			for u := range boards {
				if err := getFiles(config.ImportDir, &u); err != nil {
					return err
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case files <- u:
				}
			}

			return nil
		})
	}

	nWorkers3 := 5

	for i := 0; i < nWorkers3; i++ {
		g.Go(func() error {
			for daUser := range files {
				for bname, bfiles := range daUser.Boards {
					if err := saveBoard(daUser.Username, bname, bfiles); err != nil {
						return err
					}
				}
			}
			return nil
		})
	}

	return g.Wait()
}

func saveBoard(username string, boardname string, pictures []string) error {
	log.Info().
		Str("method", "saveBoard").
		Msgf("Importing: %s:%s\n", username, boardname)

	conn, err := redisClient.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	for _, pic := range pictures {
		if err := redisClient.AddPicture(conn, fmt.Sprintf("%s:%s", username, boardname), pic); err != nil {
			return err
		}
	}

	return nil
}

func getFiles(dir string, u *User) error {
	for boardname := range u.Boards {
		tmpFiles, err := ioutil.ReadDir(fmt.Sprintf("%s/%s/%s", dir, u.Username, boardname))
		if err != nil {
			return err
		}

		var files []string

		for _, file := range tmpFiles {
			files = append(files, file.Name())
		}

		u.Boards[boardname] = *&files
	}

	return nil
}

func getBoards(dir string, u *User) error {
	directories, err := ioutil.ReadDir(fmt.Sprintf("%s/%s", dir, u.Username))
	if err != nil {
		return err
	}

	u.Boards = make(map[string][]string)

	for _, file := range directories {
		if file.IsDir() {
			u.Boards[file.Name()] = []string{}
		}
	}

	return nil
}

func getAllUser(dir string) ([]User, error) {
	var users []User

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			u := User{Username: file.Name()}
			users = append(users, u)
		}
	}

	return users, nil
}
