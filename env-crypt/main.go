package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/presbrey/argon2aes"
	"github.com/presbrey/argon2aes/pkg/base92"
)

var (
	flagGlobal   = flag.Bool("global", false, "Walk up the directory tree to find .env files (env: $ENV_GLOBAL)")
	flagPassword = flag.String("password", "", "Password to encrypt the environment variables (env: $ENV_PASSWORD)")
	flagWrap     = flag.Int("wrap", 80, "Wrap the output at this many characters")

	skipPrefixes = []string{
		"#", "_",

		"AIDER_", "COLOR", "ENV_", "HOMEBREW_", "ITERM_", "LANG", "LC_", "LESS", "LOGNAME",
		"LS_COLORS", "MAKE", "NVM_", "PKG_", "PYENV_", "SSH_", "TERM_", "TERMINFO_", "XPC_",
	}
	skipExact = []string{
		"CPPFLAGS", "COMMAND_MODE", "HOME", "INFOPATH", "LaunchInstanceID", "LANG", "LDFLAGS", "MANPATH",
		"OLDPWD", "PATH", "PWD", "SECURITYSESSIONID", "SHELL", "SHLVL", "TERM", "TMPDIR", "USER",
	}
)

func init() {
	godotenv.Load()
	flag.Parse()

	// log lines will include file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// prefer cgo resolver
	net.DefaultResolver.PreferGo = false

	switch strings.ToLower(os.Getenv("ENV_GLOBAL")) {
	case "false", "0", "no", "off":
		*flagGlobal = false
	case "true", "1", "yes", "on":
		*flagGlobal = true
	}
	if *flagPassword == "" {
		*flagPassword = os.Getenv("ENV_PASSWORD")
	}
}

func getEnvFilePaths() ([]string, error) {
	var envFiles []string
	var lastParent string = "last"
	var nextParent string

	// Get the current working directory
	nextParent, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Start from the current directory and move up
	for lastParent != nextParent {
		// Construct the path to the .env file in the current directory
		envPath := filepath.Join(nextParent, ".env")

		// Check if the file exists
		if _, err := os.Stat(envPath); err == nil {
			// If it exists, put it on the front
			envFiles = append([]string{envPath}, envFiles...)
			if !*flagGlobal {
				break
			}
		}

		// Move to the parent directory
		lastParent = nextParent
		nextParent = filepath.Dir(lastParent)
	}
	return envFiles, nil
}

func buildEnvMap() map[string]string {
	envMap := make(map[string]string)
	for _, envVar := range os.Environ() {
		pair := strings.SplitN(envVar, "=", 2)
		if len(pair) != 2 {
			continue
		}
		for _, exact := range skipExact {
			if pair[0] == exact {
				goto skip
			}
		}
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(pair[0], prefix) {
				goto skip
			}
		}

		envMap[pair[0]] = pair[1]
	skip:
	}
	return envMap
}

func jsonEnvMap() ([]byte, error) {
	envMap := buildEnvMap()
	return json.Marshal(envMap)
}

func main() {
	envFiles, err := getEnvFilePaths()
	if err != nil {
		panic(err)
	}
	if len(envFiles) > 0 {
		err = godotenv.Overload(envFiles...)
		if err != nil {
			panic(err)
		}
	}

	envJSON, err := jsonEnvMap()
	if err != nil {
		log.Fatal(err)
	}

	ciphertext, err := argon2aes.Encrypt(envJSON, []byte(*flagPassword))
	if err != nil {
		log.Fatal(err)
	}
	base92text := base92.DefaultEncoding.EncodeToString(ciphertext)
	if *flagWrap == 0 {
		fmt.Println(base92text)
		return
	}

	// print the text at 80 characters per line
	for len(base92text) > 0 {
		if len(base92text) > *flagWrap {
			fmt.Println(base92text[:*flagWrap])
			base92text = base92text[*flagWrap:]
		} else {
			fmt.Println(base92text)
			break
		}
	}
}
