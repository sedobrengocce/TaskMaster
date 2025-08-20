package env

import "github.com/profclems/go-dotenv"

type Env struct {
	dotenv *dotenv.DotEnv
}

// New creates a new Env instance.
func ReadEnv(path *string) (*Env, error) {
	env := dotenv.New()
	if path != nil {
		env.SetConfigFile(*path)
	}
	error := dotenv.Load()
	if error != nil {
		return nil, error
	}
	return &Env{
		dotenv: env,
	}, nil
}

func (e *Env) GetDBName() string {
	return e.dotenv.GetString("DB_NAME")
} 

func (e *Env) GetDBUser() string {
	return e.dotenv.GetString("DB_USER")
}

func (e *Env) GetDBPassword() string {
	return e.dotenv.GetString("DB_PASSWORD")
}


