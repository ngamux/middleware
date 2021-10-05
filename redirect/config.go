package redirect

type Config struct {
	Rewrite Rewrite
}

func buildConfig(config Config) Config {
	if config.Rewrite == nil {
		config.Rewrite = Rewrite{}
	}
	return config
}
