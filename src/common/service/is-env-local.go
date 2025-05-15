package common_service

import "github.com/Astervia/wacraft-server/src/config/env"

func IsEnvLocal() bool {
	return env.Env == "local"
}
