package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	v1 "dex/v1"
	v2 "dex/v2"
)

// Paths to search for dex files.
var configFileLocations = []string{"dex.yaml", "dex.yml", ".dex.yaml", ".dex.yml"}

/*
1. Try to locate a dex file, throw an error and exit if there is no config file.
2. Load the content of the dex file
3. Attempt to parse the dex file as v1 and then v2 YAML.
*/
func main() {

	/* Find the name of the dex file we're using. */
	if filename, err := findConfigFile(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		/* Load the raw yaml data */
	} else if dexData, err := loadDexFile(filename); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		/* Attempt parsing as v1 */
	} else if dexFile, err := v1.ParseConfig(dexData); err == nil {
		v1.Run(dexFile, os.Args)
		/* Attempt parsing as v2 */
	} else if dexFile, err := v2.ParseConfig(dexData); err == nil {
		v2.Run(dexFile, os.Args)
		/* failure */
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadDexFile(filename string) ([]byte, error) {

	if fileContent, err := os.Open(filename); err != nil {
		return []byte{}, fmt.Errorf("yamlFile.Get err #%s", err)
	} else if dexData, err := io.ReadAll(fileContent); err != nil {
		return []byte{}, fmt.Errorf("yamlFile.Get err #%v ", err)
	} else {
		return dexData, err
	}
}

/*
Search through the config_files array and return the first
dex file that exists.
*/
func findConfigFile() (string, error) {

	/* If the first block parameter is "~~", this parameter
	       is removed and we check for dex files in the users
		   home directory instead of the current working directory
	*/
	useHome := false
	if len(os.Args) > 1 && os.Args[1] == "~~" {
		os.Args = os.Args[1:]
		useHome = true
	}

	homeDir, err := os.UserHomeDir()
	if useHome && err != nil {
		fmt.Fprintf(os.Stderr, "error finding home directory: %v", err)
	}

	/* DEX_FILE environment variable takes priority. If ~~ was set
	   then we check for the DEX_FILE path relative to the users
	   home directory.
	*/
	if dexFileEnv := os.Getenv("DEX_FILE"); len(dexFileEnv) > 0 {
		if useHome {
			dexFileEnv = filepath.Join(homeDir, dexFileEnv)
		}

		if _, err := os.Stat(dexFileEnv); err == nil {
			return dexFileEnv, nil
		}
	}

	for _, filename := range configFileLocations {

		if useHome {
			filename = filepath.Join(homeDir, filename)
		}

		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}
	}

	return "", fmt.Errorf("no dex file was found.  Searched %v", configFileLocations)
}
