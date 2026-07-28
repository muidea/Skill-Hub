package configport

import "github.com/muidea/skill-hub/internal/config"

type RepositoryConfig = config.RepositoryConfig

type Config interface {
	RootDir() (string, error)
	Initialized() error
}
