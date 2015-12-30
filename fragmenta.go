// A command line tool for fragmenta which can be used to build and run websites
// this tool calls subcommands for most of the work, usually one command per file in this pkg
// See docs at http://godoc.org/github.com/fragmenta/fragmenta

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

const (
	// The version of this tool
	fragmentaVersion = "1.3.1"

	// Used for outputting console messages
	fragmentaDivider = "\n------\n"
)

var (
	// ConfigDevelopment holds the development config from fragmenta.json
	ConfigDevelopment map[string]string

	// ConfigProduction holds development config from fragmenta.json
	ConfigProduction map[string]string

	// ConfigTest holds the app test config from fragmenta.json
	ConfigTest map[string]string
)

// main - Parse the command line arguments and respond
func main() {

	log.SetFlags(log.Ltime)

	args := os.Args
	command := ""

	if len(args) > 1 {
		command = args[1]
	}

	// We should intelligently read project path depending on the command?
	// Or just assume we act on the current directory?
	// NB projectPath might be different from the path in config, which MUST be within a GOPATH
	// this is the local project path
	projectPath, err := filepath.Abs(".")
	if err != nil {
		log.Printf("Error getting path %s", err)
		return
	}

	// If this is a valid fragmenta project, try reading the config
	// NB we still run even if config fails, as we want to at least try a build/run cycle to enable bootstrap
	if isValidProject(projectPath) {
		readConfig(projectPath)
	}

	switch command {

	case "new", "n":
		RunNew(args)

	case "version", "v":
		ShowVersion()

	case "help", "h", "wat", "?":
		ShowHelp(args)

	case "server", "s":
		if requireValidProject(projectPath) {
			RunServer(projectPath)
		}

	case "test", "t":
		if requireValidProject(projectPath) {
			RunTests(args)
		}

	case "build", "B":
		if requireValidProject(projectPath) {
			RunBuild(args)
		}

	case "generate", "g":
		if requireValidProject(projectPath) {
			RunGenerate(args)
		}

	case "migrate", "m":
		if requireValidProject(projectPath) {
			RunMigrate(args)
		}

	case "backup", "b":
		if requireValidProject(projectPath) {
			RunBackup(args)
		}

	case "restore", "r":
		if requireValidProject(projectPath) {
			RunRestore(args)
		}

	case "deploy", "d":
		if requireValidProject(projectPath) {
			RunDeploy(args)
		}

	default:
		if requireValidProject(projectPath) {
			RunServer(projectPath)
		} else {
			ShowHelp(args)
		}
	}

}

// ShowVersion shows the version of this tool
func ShowVersion() {
	helpString := fragmentaDivider
	helpString += fmt.Sprintf("Fragmenta version: %s", fragmentaVersion)
	helpString += fragmentaDivider
	log.Print(helpString)
}

// ShowHelp shows the help for this tool
func ShowHelp(args []string) {
	helpString := fragmentaDivider
	helpString += fmt.Sprintf("Fragmenta version: %s", fragmentaVersion)
	helpString += "\n  fragmenta version -> display version"
	helpString += "\n  fragmenta help -> display help"
	helpString += "\n  fragmenta new [app|cms|blog|URL] path/to/app -> creates a new app from the repository at URL at the path supplied"
	helpString += "\n  fragmenta -> builds and runs a fragmenta app"
	helpString += "\n  fragmenta server -> builds and runs a fragmenta app"
	helpString += "\n  fragmenta test  -> run tests"
	helpString += "\n  fragmenta migrate -> runs new sql migrations in db/migrate"
	helpString += "\n  fragmenta backup [development|production|test] -> backup the database to db/backup"
	helpString += "\n  fragmenta restore [development|production|test] -> backup the database from latest file in db/backup"
	helpString += "\n  fragmenta deploy [development|production|test] -> build and deploy using bin/deploy"
	helpString += "\n  fragmenta generate resource [name] [fieldname]:[fieldtype]* -> creates resource CRUD actions and views"
	helpString += "\n  fragmenta generate migration [name] -> creates a new named sql migration in db/migrate"

	helpString += fragmentaDivider
	log.Print(helpString)
}

// FIXME - move all instances of hardcoded paths out into optional app config variables
// Ideally we don't care about project structure apart from the load the fragmenta.json file

// serverName returns the name of our server file - TODO:read from config
func serverName() string {
	return "fragmenta-server" // for now, should use configs
}

func localServerPath(projectPath string) string {
	return fmt.Sprintf("%s/bin/%s-local", projectPath, serverName())
}

func serverPath(projectPath string) string {
	return fmt.Sprintf("%s/bin/%s", projectPath, serverName())
}

func serverCompilePath(projectPath string) string {
	return path.Join(projectPath, "server.go")
}

// Return the src to scan assets for compilation
// Can this be set within the project setup instead to avoid hardcoding here?
func srcPath(projectPath string) string {
	return projectPath + "src"
}

func publicPath(projectPath string) string {
	return projectPath + "public"
}

func configPath(projectPath string) string {
	return projectPath + "/secrets/fragmenta.json"
}

func secretsPath(projectPath string) string {
	return projectPath + "/secrets"
}

func templatesPath() string {
	return os.ExpandEnv("$GOPATH/src/github.com/fragmenta/fragmenta/templates")
}

// RunServer runs the server
func RunServer(projectPath string) {
	ShowVersion()

	killServer()

	log.Println("Building server...")
	err := buildServer(localServerPath(projectPath), nil)

	if err != nil {
		log.Printf("Error building server: %s", err)
		return
	}

	log.Println("Launching server...")
	cmd := exec.Command(localServerPath(projectPath))
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	err = cmd.Start()
	if err != nil {
		log.Println(err)
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	cmd.Wait()

}

// killServer kills the server with a unix command - FIXME:Windows
func killServer() {
	runCommand("killall", "-9", serverName())
}

// runCommand runs a command with exec.Command
func runCommand(command string, args ...string) ([]byte, error) {

	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, err
	}

	return output, nil
}

// runCommand runs a command with exec.Command
func runCommandSetEnv(command string, p_env []string, args ...string) ([]byte, error) {
	// It seems the only way to get env vars to exec is to set them manually here
	for i := 0; i < len(p_env); i += 2 {
		os.Setenv(p_env[i], p_env[i + 1])
	}
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, err
	}

	return output, nil
}

// requireValidProject returns true if we have a valid project at projectPath
func requireValidProject(projectPath string) bool {
	if isValidProject(projectPath) {
		return true
	}

	log.Printf("\nNo fragmenta project found at this path\n")
	return false

}

// isValidProject returns true if this is a valid fragmenta project (checks for server.go file and config file)
func isValidProject(projectPath string) bool {

	// Make sure we have server.go at root of this dir
	_, err := os.Stat(serverCompilePath(projectPath))
	if err != nil {
		return false
	}

	return true
}

// fileExists returns true if this file exists
func fileExists(p string) bool {
	_, err := os.Stat(p)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return true
}

// readConfig reads our config file and set up the server accordingly
func readConfig(projectPath string) error {
	configPath := configPath(projectPath)

	// Read the config json file
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("Error opening config %s %v", configPath, err)
		return err
	}

	var data map[string]map[string]string
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Printf("Error parsing config %s %v", configPath, err)
		return err
	}

	ConfigDevelopment = data["development"]
	ConfigProduction = data["production"]
	ConfigTest = data["test"]

	return nil
}
