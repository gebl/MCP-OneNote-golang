package config

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

type Config struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	TenantID     string   `json:"tenant_id"`
	RedirectURI  string   `json:"redirect_uri"`
	Toolsets     []string `json:"toolsets"`
	NotebookName string   `json:"notebook_name"`
}

func Load() (*Config, error) {
	log.Println("[config] Loading configuration...")
	cfg := &Config{
		ClientID:     os.Getenv("ONENOTE_CLIENT_ID"),
		ClientSecret: os.Getenv("ONENOTE_CLIENT_SECRET"),
		TenantID:     os.Getenv("ONENOTE_TENANT_ID"),
		RedirectURI:  os.Getenv("ONENOTE_REDIRECT_URI"),
		NotebookName: os.Getenv("ONENOTE_NOTEBOOK_NAME"),
	}
	log.Printf("[config] Loaded from env: client_id=%q, tenant_id=%q, redirect_uri=%q, notebook_name=%q", cfg.ClientID, cfg.TenantID, cfg.RedirectURI, cfg.NotebookName)

	// Toolsets from env
	if toolsets := os.Getenv("ONENOTE_TOOLSETS"); toolsets != "" {
		cfg.Toolsets = strings.Split(toolsets, ",")
		log.Printf("[config] Loaded toolsets from env: %v", cfg.Toolsets)
	}

	// Optionally load from config file
	if path := os.Getenv("ONENOTE_MCP_CONFIG"); path != "" {
		log.Printf("[config] Loading from config file: %s", path)
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		dec := json.NewDecoder(f)
		if err := dec.Decode(cfg); err != nil {
			return nil, err
		}
		log.Printf("[config] Loaded from file: client_id=%q, tenant_id=%q, redirect_uri=%q, toolsets=%v, notebook_name=%q", cfg.ClientID, cfg.TenantID, cfg.RedirectURI, cfg.Toolsets, cfg.NotebookName)
	}

	log.Printf("[config] Final config: %+v", cfg)
	return cfg, nil
}
