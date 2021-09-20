package static

type Config struct {
	Root   string
	Prefix string
}

func makeConfig(config Config) Config {
	cfg := Config{
		Root:   "public",
		Prefix: "/public",
	}
	if config.Root != "" {
		cfg.Root = config.Root
	}
	if config.Prefix != "" {
		cfg.Prefix = config.Prefix
	}

	return cfg
}
