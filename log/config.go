package log

type Config struct {
	Format string
}

var configDefault Config = Config{
	Format: "${method} ${path} ${status}",
}
