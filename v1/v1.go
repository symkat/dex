package v1

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/goccy/go-yaml"
)

type DexFile []struct {
	Name     string   `yaml:"name"`
	Desc     string   `yaml:"desc"`
	Commands []string `yaml:"shell"`
	Children DexFile  `yaml:"children"`
}

/*
1. If there was no commands to run, display the menu of commands the DexFile knows about.
2. If there was a command to run, find it and run it.  If it's invalid, say so and display the menu.
*/
func Run(dexFile DexFile, args []string) {

	/* No commands asked for: show menu and exit */
	if len(args) == 1 {
		displayMenu(os.Stdout, dexFile, 0)
		os.Exit(0)
	}

	/* No commands were found from the arguments the user passed: show error, menu and exit */
	commands, err := resolveCmdToCodeblock(dexFile, args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: No commands were found at %v\n\nSee the menu:\n", args[1:])
		displayMenu(os.Stderr, dexFile, 0)
		os.Exit(1)
	}

	/* Found commands: run them */
	runCommands(commands)
}

/*
Attempt to parse the YAML content into DexFile format
*/
func ParseConfig(configData []byte) (DexFile, error) {

	var dexFile DexFile

	if err := yaml.Unmarshal([]byte(configData), &dexFile); err != nil {
		return nil, err
	}

	return dexFile, nil
}

/*
Display the menu by recursively processing each element of the DexFile and

	showing the name and description for the command.  Children are indented with
	4 spaces.
*/
func displayMenu(w io.Writer, dexFile DexFile, indent int) {
	for _, elem := range dexFile {

		fmt.Fprintf(w, "%s%-24v: %v\n", strings.Repeat(" ", indent*4), elem.Name, elem.Desc)

		if len(elem.Children) >= 1 {
			displayMenu(w, elem.Children, indent+1)
		}
	}
}

/*
Find the list of commands to run for a given command path.

	For example, cmd = [ 'foo', 'bar', 'blee' ] would check if 'foo' is a valid command,
	then call itself with the child DexFile of foo, and cmd = ['bar', 'blee'].  Then bar's
	child DexFile would be called with [ 'blee' ] and return the list of commands.
*/
func resolveCmdToCodeblock(dexFile DexFile, cmds []string) ([]string, error) {
	for _, elem := range dexFile {
		if elem.Name == cmds[0] {
			if len(cmds) >= 2 {
				return resolveCmdToCodeblock(elem.Children, cmds[1:])
			} else {
				return elem.Commands, nil
			}
		}
	}
	return []string{}, errors.New("could not find command")
}

/*
Given a list of commands, run them.

	Uses bash so that quoting, shell expansion, etc works.
	Writes the stdout/stderr as one would expect.
*/
func runCommands(commands []string) {
	for _, command := range commands {
		cmd := exec.Command("/bin/bash", "-c", command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to run command: ", err)
		}
	}
}
