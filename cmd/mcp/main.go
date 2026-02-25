package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dreamzero-oxm/go-react-agent/mcp"
)

// Command represents a CLI command.
type Command struct {
	Name        string
	Description string
	Func        func() error
}

var commands = map[string]*Command{}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmdName := os.Args[1]
	cmd, exists := commands[cmdName]

	if !exists {
		fmt.Printf("Unknown command: %s\n\n", cmdName)
		printUsage()
		os.Exit(1)
	}

	if err := cmd.Func(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// printUsage prints the usage information for the CLI tool.
func printUsage() {
	fmt.Println("go-react-agent MCP Management Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mcp-cmd <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	for name, cmd := range commands {
		fmt.Printf("  %-20s %s\n", name, cmd.Description)
	}
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mcp-cmd add --name filesystem --command npx --args \"-y,@modelcontextprotocol/server-filesystem,/path\"")
	fmt.Println("  mcp-cmd list")
	fmt.Println("  mcp-cmd status")
}

// registerCommand registers a new command with the CLI.
//
// Parameters:
//   - name: The command name.
//   - description: A short description of the command.
//   - fn: The function to execute when the command is called.
func registerCommand(name, description string, fn func() error) {
	cmd := &Command{
		Name:        name,
		Description: description,
		Func:        fn,
	}
	commands[name] = cmd
}

func init() {
	registerCommand("add", "Add a new MCP server", handleAdd)
	registerCommand("list", "List all configured MCP servers", handleList)
	registerCommand("enable", "Enable a disabled MCP server", handleEnable)
	registerCommand("disable", "Disable an MCP server", handleDisable)
	registerCommand("remove", "Remove an MCP server", handleRemove)
	registerCommand("status", "Show status of all MCP servers", handleStatus)
}

// handleAdd handles the 'add' command.
//
// Returns:
//   - error: An error if adding the server fails.
func handleAdd() error {
	flagSet := flag.NewFlagSet("add", flag.ExitOnError)
	name := flagSet.String("name", "", "Server name (required)")
	command := flagSet.String("command", "", "Command to run (for stdio)")
	args := flagSet.String("args", "", "Comma-separated arguments")
	env := flagSet.String("env", "", "Comma-separated KEY=VALUE environment variables")
	transport := flagSet.String("transport", "stdio", "Transport type: stdio or sse")
	url := flagSet.String("url", "", "URL (for sse)")
	headers := flagSet.String("headers", "", "Comma-separated KEY:VALUE headers")
	timeout := flagSet.Int("timeout", 30, "Timeout in seconds")
	disabled := flagSet.Bool("disabled", false, "Add server in disabled state")

	flagSet.Parse(os.Args[2:])

	if *name == "" {
		return fmt.Errorf("server name is required")
	}

	config, err := mcp.LoadConfig("~/.go-react-agent/mcp/mcp.json", ".go-react-agent/mcp/mcp.json")
	if err != nil {
		config = mcp.NewDefaultConfig()
	}

	serverConfig := mcp.ServerConfig{
		Command:   *command,
		Args:      mcp.ParseArgs(*args),
		Env:       mcp.ParseEnvMap(*env),
		Transport: *transport,
		URL:       *url,
		Headers:   mcp.ParseHeaderMap(*headers),
		Timeout:   *timeout,
		Disabled:  *disabled,
	}

	if err := config.AddServer(*name, serverConfig); err != nil {
		return err
	}

	if err := config.Save(); err != nil {
		return err
	}

	fmt.Printf("Successfully added MCP server '%s'\n", *name)
	return nil
}

// handleList handles the 'list' command.
//
// Returns:
//   - error: An error if listing servers fails.
func handleList() error {
	config, err := mcp.LoadConfig("~/.go-react-agent/mcp.json", ".go-react-agent/mcp.json")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	servers := config.ListServers()

	if len(servers) == 0 {
		fmt.Println("No MCP servers configured.")
		return nil
	}

	fmt.Println("Configured MCP Servers:")
	fmt.Println()

	for _, name := range servers {
		server, _ := config.GetServer(name)
		status := "enabled"
		if server.Disabled {
			status = "disabled"
		}

		fmt.Printf("  • %s (%s)\n", name, status)

		if server.Command != "" {
			fmt.Printf("      Command: %s %s\n", server.Command, strings.Join(server.Args, " "))
		}
		if server.URL != "" {
			fmt.Printf("      URL: %s\n", server.URL)
		}
	}

	return nil
}

// handleEnable handles the 'enable' command.
//
// Returns:
//   - error: An error if enabling the server fails.
func handleEnable() error {
	flagSet := flag.NewFlagSet("enable", flag.ExitOnError)
	name := flagSet.String("name", "", "Server name (required)")
	flagSet.Parse(os.Args[2:])

	if *name == "" {
		return fmt.Errorf("server name is required")
	}

	config, err := mcp.LoadConfig("~/.go-react-agent/mcp.json", ".go-react-agent/mcp.json")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.EnableServer(*name); err != nil {
		return err
	}

	if err := config.Save(); err != nil {
		return err
	}

	fmt.Printf("Enabled MCP server '%s'\n", *name)
	return nil
}

// handleDisable handles the 'disable' command.
//
// Returns:
//   - error: An error if disabling the server fails.
func handleDisable() error {
	flagSet := flag.NewFlagSet("disable", flag.ExitOnError)
	name := flagSet.String("name", "", "Server name (required)")
	flagSet.Parse(os.Args[2:])

	if *name == "" {
		return fmt.Errorf("server name is required")
	}

	config, err := mcp.LoadConfig("~/.go-react-agent/mcp.json", ".go-react-agent/mcp.json")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.DisableServer(*name); err != nil {
		return err
	}

	if err := config.Save(); err != nil {
		return err
	}

	fmt.Printf("Disabled MCP server '%s'\n", *name)
	return nil
}

// handleRemove handles the 'remove' command.
//
// Returns:
//   - error: An error if removing the server fails.
func handleRemove() error {
	flagSet := flag.NewFlagSet("remove", flag.ExitOnError)
	name := flagSet.String("name", "", "Server name (required)")
	flagSet.Parse(os.Args[2:])

	if *name == "" {
		return fmt.Errorf("server name is required")
	}

	config, err := mcp.LoadConfig("~/.go-react-agent/mcp.json", ".go-react-agent/mcp.json")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.RemoveServer(*name); err != nil {
		return err
	}

	if err := config.Save(); err != nil {
		return err
	}

	fmt.Printf("Removed MCP server '%s'\n", *name)
	return nil
}

// handleStatus handles the 'status' command.
//
// Returns:
//   - error: An error if checking status fails.
func handleStatus() error {
	flagSet := flag.NewFlagSet("status", flag.ExitOnError)
	debug := flagSet.Bool("debug", false, "Enable debug logging")
	flagSet.Parse(os.Args[2:])

	config, err := mcp.LoadConfig("~/.go-react-agent/mcp.json", ".go-react-agent/mcp.json")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	manager := mcp.NewManager(config)
	manager.SetDebug(*debug)

	if err := manager.Start(); err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}
	defer manager.Stop()

	statuses := manager.GetStatus()

	if len(statuses) == 0 {
		fmt.Println("No MCP servers configured.")
		return nil
	}

	for _, status := range statuses {
		var icon string
		var statusStr string

		switch status.Status {
		case "running":
			icon = "✅"
			statusStr = "Running"
		case "disabled":
			icon = "⚪"
			statusStr = "Disabled"
		case "failed":
			icon = "❌"
			statusStr = "Failed"
		case "initializing":
			icon = "⏳"
			statusStr = "Initializing"
		}

		fmt.Printf("%s %s (%s)\n", icon, status.Name, status.Type)
		fmt.Printf("   Status: %s\n", statusStr)

		if status.Command != "" {
			fmt.Printf("   Command: %s\n", status.Command)
		}
		if status.URL != "" {
			fmt.Printf("   URL: %s\n", status.URL)
		}

		if status.Error != "" {
			fmt.Printf("   Error: %s\n", status.Error)
		}

		if len(status.Tools) > 0 {
			fmt.Printf("   Tools: %d available\n", len(status.Tools))
			for _, tool := range status.Tools {
				fmt.Printf("     - %s: %s\n", tool.Name, tool.Description)
			}
		}

		if len(status.Resources) > 0 {
			fmt.Printf("   Resources: %d available\n", len(status.Resources))
			for _, resource := range status.Resources {
				fmt.Printf("     - %s: %s\n", resource.URI, resource.Name)
			}
		}

		if len(status.Prompts) > 0 {
			fmt.Printf("   Prompts: %d available\n", len(status.Prompts))
			for _, prompt := range status.Prompts {
				fmt.Printf("     - %s: %s\n", prompt.Name, prompt.Description)
			}
		}

		fmt.Println()
	}

	return nil
}
