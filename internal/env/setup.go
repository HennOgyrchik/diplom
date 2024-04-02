package env

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jlaffaye/ftp"
	"github.com/sethvargo/go-envconfig"
	"my_fund/internal/db"
	"my_fund/internal/env/config"
	"my_fund/internal/fileStorage"
)

type Env struct {
	DB    *db.Repository
	FTP   *fileStorage.FileStorage
	Token string
}

func Setup(ctx context.Context) (*Env, error) {
	var cfg config.Config
	env := &Env{}

	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("env processing: %w", err)
	}

	env.Token = cfg.Token

	usersDBConn, err := pgxpool.Connect(ctx, cfg.Postgres.ConnectionURL())
	if err != nil {
		return nil, fmt.Errorf("pgxpool Connect: %w", err)
	}

	env.DB = db.New(usersDBConn, cfg.Postgres.DBTimeout)

	ftpClient, err := ftp.Dial(cfg.FTP.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("FTP Connect: %w", err)
	}

	if err = ftpClient.Login(cfg.FTP.User, cfg.FTP.Password); err != nil {
		return nil, fmt.Errorf("FTP Login: %w", err)
	}

	env.FTP = fileStorage.New(ftpClient)

	return env, nil
}
