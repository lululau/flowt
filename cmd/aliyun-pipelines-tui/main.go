package main

import (
	"aliyun-pipelines-tui/internal/api" // Import the api package
	"aliyun-pipelines-tui/internal/ui"  // Local package for UI components
	"fmt"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// 云效服务接入点域名
	Endpoint string `yaml:"endpoint"`
	// 个人访问令牌 (推荐的认证方式)
	PersonalAccessToken string `yaml:"personal_access_token"`
	// 企业 ID（组织 ID）
	OrganizationID string `yaml:"organization_id"`
	// AccessKey 认证方式 (备用方式)
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	RegionID        string `yaml:"region_id"`
	// 编辑器和分页器配置
	Editor string `yaml:"editor,omitempty"`
	Pager  string `yaml:"pager,omitempty"`
}

// loadConfig loads configuration from ~/.config/flowt.yml
func loadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "flowt.yml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found at %s", configPath)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.OrganizationID == "" {
		return fmt.Errorf("organization_id is required in configuration")
	}

	// 检查认证方式：优先使用个人访问令牌，其次使用AccessKey
	hasPersonalToken := config.PersonalAccessToken != ""
	hasAccessKey := config.AccessKeyID != "" && config.AccessKeySecret != ""

	if !hasPersonalToken && !hasAccessKey {
		return fmt.Errorf("either personal_access_token or both access_key_id and access_key_secret are required")
	}

	return nil
}

// GetEditor returns the editor command to use, following the priority:
// 1. Config file "editor" field
// 2. VISUAL environment variable
// 3. EDITOR environment variable
// 4. Default to "vim"
func GetEditor(config *Config) string {
	// First check config file
	if config.Editor != "" {
		return config.Editor
	}

	// Then check VISUAL environment variable
	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}

	// Then check EDITOR environment variable
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Default to vim
	return "vim"
}

// GetPager returns the pager command to use, following the priority:
// 1. Config file "pager" field
// 2. PAGER environment variable
// 3. Default to "less"
func GetPager(config *Config) string {
	// First check config file
	if config.Pager != "" {
		return config.Pager
	}

	// Then check PAGER environment variable
	if pager := os.Getenv("PAGER"); pager != "" {
		return pager
	}

	// Default to less
	return "less"
}

func main() {
	// Load configuration from file
	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nPlease create a configuration file at ~/.config/flowt.yml with the following format:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "# 企业 ID（组织 ID）- 必填")
		fmt.Fprintln(os.Stderr, "organization_id: your_organization_id")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "# 推荐：使用个人访问令牌认证")
		fmt.Fprintln(os.Stderr, "personal_access_token: your_personal_access_token")
		fmt.Fprintln(os.Stderr, "endpoint: openapi-rdc.aliyuncs.com  # 可选，默认值")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "# 或者：使用AccessKey认证（备用方式）")
		fmt.Fprintln(os.Stderr, "# access_key_id: your_access_key_id")
		fmt.Fprintln(os.Stderr, "# access_key_secret: your_access_key_secret")
		fmt.Fprintln(os.Stderr, "# region_id: cn-hangzhou  # 可选，默认值")
		os.Exit(1)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation error: %v\n", err)
		os.Exit(1)
	}

	// Initialize API client with configuration
	var apiClient *api.Client

	// 优先使用个人访问令牌认证
	if config.PersonalAccessToken != "" {
		endpoint := config.Endpoint
		if endpoint == "" {
			endpoint = "openapi-rdc.aliyuncs.com" // 默认端点
		}
		var err error
		apiClient, err = api.NewClientWithToken(endpoint, config.PersonalAccessToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing API client with personal access token: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 使用AccessKey认证作为备用方式
		regionID := config.RegionID
		if regionID == "" {
			regionID = "cn-hangzhou" // 默认区域
		}
		var err error
		apiClient, err = api.NewClient(config.AccessKeyID, config.AccessKeySecret, regionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing API client with access key: %v\n", err)
			os.Exit(1)
		}
	}

	// Set transparent background style
	tcell.StyleDefault = tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorDefault)

	// Initialize tview.Application
	app := tview.NewApplication()

	// Set global config for UI components
	ui.SetGlobalConfig(GetEditor(config), GetPager(config))

	// Create the main view (Pages) using ui.NewMainView()
	mainPages := ui.NewMainView(app, apiClient, config.OrganizationID) // Pass apiClient and orgId

	// Set up global input capture for 'q' and Ctrl+C to stop the application
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'Q' {
				app.Stop()
				return nil
			}
		}
		return event
	})

	// Set the root of the application and run
	if err := app.SetRoot(mainPages, true).EnableMouse(true).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}
