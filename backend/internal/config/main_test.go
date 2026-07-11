package config

import "testing"

func TestValidateAllowsLocalMiniappBrowserUserOnlyInLocalEnvironment(t *testing.T) {
	localConfig := validConfigForTest()
	localConfig.Env = "local"
	localConfig.API.MiniappLocalTelegramUserID = 42
	if err := validate(localConfig); err != nil {
		t.Fatalf("validate local config: %v", err)
	}

	productionConfig := validConfigForTest()
	productionConfig.Env = "production"
	productionConfig.API.MiniappLocalTelegramUserID = 42
	if err := validate(productionConfig); err == nil {
		t.Fatal("expected non-local Mini App browser user to be rejected")
	}
}

func TestValidateRejectsNegativeLocalMiniappBrowserUserID(t *testing.T) {
	cfg := validConfigForTest()
	cfg.Env = "local"
	cfg.API.MiniappLocalTelegramUserID = -1
	if err := validate(cfg); err == nil {
		t.Fatal("expected negative Mini App browser user ID to be rejected")
	}
}

func validConfigForTest() *Config {
	return &Config{
		DB: DBConfig{
			Host:     "postgres",
			Name:     "gravel_bot",
			User:     "gravel",
			Password: "password",
		},
		Bot: BotConfig{Token: "token"},
		API: APIConfig{JWTSecret: "secret"},
	}
}
