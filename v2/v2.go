package v2

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
)

type VarCfg struct {
	StringValue string
	ListValue   []string
	FromCommand string
	FromEnv     string
	Default     string
}

func (varCfg VarCfg) Value() (any, error) {

	if len(varCfg.StringValue) > 0 {
		return varCfg.Value, nil
	} else if len(varCfg.ListValue) > 0 {
		return varCfg.ListValue, nil
	}

	return nil, errors.New("undefined")
}

type VarValue interface {
	string | []string
}

func SetVarValue[Value VarValue](varCfg *VarCfg, value Value) error {
	switch typeValue := any(value).(type) {
	case []string:
		varCfg.ListValue = typeValue

	case string:
		varCfg.StringValue = typeValue
	default:
		return errors.New("unknown VarCfg value type")
	}

	return nil
}

type Command struct {
	Exec      string
	Diag      string
	Dir       string
	ForVars   []string
	Shell     string
	ShellArgs []string
	Condition string
}

type Block struct {
	Name        string           `yaml:"name"`
	Desc        string           `yaml:"desc"`
	CommandsRaw []map[string]any `yaml:"commands"`
	Commands    []Command        `yaml:"Commands"`
	Vars        map[string]any   `yaml:"vars"`
	Dir         string           `yaml:"dir"`
	Shell       string           `yaml:"shell"`
	ShellArgs   []string         `yaml:"shell_args"`
	Children    []Block          `yaml:"children"`
}
type DexFile2 struct {
	Version   int            `yaml:"version"`
	Vars      map[string]any `yaml:"vars"`
	Blocks    []Block        `yaml:"blocks"`
	Shell     string         `yaml:"shell"`
	ShellArgs []string       `yaml:"shell_args"`
}

var DefaultShell = "/bin/bash"
var DefaultShellArgs = []string{"-c"}
var VarCfgs = map[string]VarCfg{}

/* Helper function to set default value if field value is unset */
func checkSetDefault[D VarValue](field *D, def D) {

	if len(*field) == 0 {
		*field = def
	}
}

/* Helper function to set field value if override value is set */
func checkSetOverride[D VarValue](field *D, override D) {

	if len(override) != 0 {
		*field = override
	}
}

/*
Attempt to parse the YAML content into DexFile2 format
and do some sanity checks and set defaults.
*/
func ParseConfig(configData []byte) (DexFile2, error) {

	var dexFile DexFile2

	if err := yaml.Unmarshal([]byte(configData), &dexFile); err != nil {
		return DexFile2{}, err
	} else if dexFile.Version != 2 {
		return DexFile2{}, errors.New("incorrect version number")
	}

	checkSetDefault(&dexFile.Shell, DefaultShell)
	checkSetDefault(&dexFile.ShellArgs, DefaultShellArgs)

	return dexFile, nil
}

/*
1. If there was no commands to run, display the menu of commands the DexFile knows about.
2. If there was a command to run, find it and run it.  If it's invalid, say so and display the menu.
*/
func Run(dexFile DexFile2, args []string) {

	/* No commands asked for: show menu and exit */

	if len(args) == 1 {
		displayMenu(os.Stdout, dexFile.Blocks, 0)
		os.Exit(0)
	}

	initVars(dexFile.Vars)

	block, err := initBlockFromPath(dexFile, args[1:])

	/* No commands were found from the arguments the user passed: show error, menu and exit */
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		displayMenu(os.Stderr, dexFile.Blocks, 0)
		os.Exit(1)
	}

	config := ExecConfig{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	processBlock(block, config)
}

func initBlockFromPath(dexFile DexFile2, blockPath []string) (Block, error) {

	block, err := resolveCmdToCodeblock(dexFile.Blocks, blockPath)

	if err != nil {
		return Block{}, fmt.Errorf("error: No commands were found at %v\n\nSee the menu", blockPath[1:])
	}

	/* Found block.  Init variables, set defaults and process the
	   block and its commands */
	checkSetDefault(&block.Shell, dexFile.Shell)
	checkSetDefault(&block.ShellArgs, dexFile.ShellArgs)
	initVars(block.Vars)
	initBlockCommands(&block)

	return block, nil
}

/*
Display the menu by recursively processing each element of the DexFile and
showing the name and description for the command.  Children are indented with
4 spaces.
*/
func displayMenu(w io.Writer, blocks []Block, indent int) {
	for _, elem := range blocks {

		fmt.Fprintf(w, "%s%-24v: %v\n", strings.Repeat(" ", indent*4), elem.Name, elem.Desc)

		if len(elem.Children) >= 1 {
			displayMenu(w, elem.Children, indent+1)
		}
	}
}

func resolveCmdToCodeblock(blocks []Block, cmds []string) (Block, error) {

	for _, elem := range blocks {
		if elem.Name == cmds[0] {
			if len(cmds) >= 2 {
				return resolveCmdToCodeblock(elem.Children, cmds[1:])
			} else {
				return elem, nil
			}
		}
	}
	return Block{}, errors.New("could not find command")
}

/* helper function that checks multiple keys for value */
func checkKeys[T VarValue](cfg map[string]any, keys []string) (T, bool) {
	var empty T

	for _, key := range keys {
		if cfg[key] != nil {
			return cfg[key].(T), true
		}
	}

	return empty, false
}

func initVars(varMap map[string]any) {
	for varName, value := range varMap {

		switch typeVal := value.(type) {
		/* VarCfg */
		case map[string]any:

			varCfg := VarCfg{}

			if fromEnv, ok := checkKeys[string](typeVal, []string{"from-env", "from_env"}); ok {
				varCfg.FromEnv = fromEnv
				if envVal := os.Getenv(varCfg.FromEnv); len(envVal) > 0 {
					varCfg.StringValue = envVal
				}
			}

			if fromCommand, ok := checkKeys[string](typeVal, []string{"from-command", "from_command"}); ok {

				varCfg.FromCommand = fromCommand

				var output bytes.Buffer

				execConfig := ExecConfig{
					Stdout: &output,
				}

				/* TODO? Allow setting custom shell for this.
				   Would be a just convenience since you already
				   do something like:
				   from_command: '/usr/bin/zsh -c "echo hello"'
				*/
				execConfig.Cmd = "/bin/bash"
				execConfig.Args = []string{"-c", varCfg.FromCommand}

				if exit := execCommand(execConfig); exit == 0 {
					lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")

					/* Turn multi-line output into List */
					if len(lines) > 1 {

						SetVarValue(&varCfg, lines)
					} else {
						SetVarValue(&varCfg, lines[0])
					}
				}
			}

			if typeVal["default"] != nil {
				varCfg.Default = typeVal["default"].(string)
			}

			if _, err := varCfg.Value(); err != nil && len(varCfg.Default) > 0 {

				SetVarValue(&varCfg, varCfg.Default)
			}

			VarCfgs[varName] = varCfg

		/* List */
		case []any:

			VarCfgs[varName] = VarCfg{
				ListValue: []string{},
			}

			for _, elem := range typeVal {

				entry := VarCfgs[varName]
				SetVarValue(&entry, append(entry.ListValue, elem.(string)))

				VarCfgs[varName] = entry
			}

		/* Integer */
		case uint64:

			VarCfgs[varName] = VarCfg{
				StringValue: strconv.FormatUint(typeVal, 10),
			}

		/* String */
		case string:

			VarCfgs[varName] = VarCfg{
				StringValue: typeVal,
			}
		default:
			fmt.Printf("I don't know about type %T for %s!\n", typeVal, varName)
		}
	}
}

/* Capture the variable name inside the perl template delimiters */
var fixupRe = regexp.MustCompile(`\[%\s*([^\s%]+)\s*%\]`)
var tt = template.New("variable_parser")

func render(tmpl string, varCfgs map[string]VarCfg) string {

	if len(tmpl) == 0 {
		return ""
	}

	/*
	   Converting from the template format established in the perl version
	*/
	t1, err := tt.Parse(fixupRe.ReplaceAllString(tmpl, "{{ .$1.StringValue }}"))
	if err != nil {
		panic(err)
	}

	var renderBuf bytes.Buffer

	t1.Execute(&renderBuf, varCfgs)

	return renderBuf.String()
}

func assignIfSet[T string | []string](commandCfg map[string]any, key string, field *T) {
	if commandCfg[key] != nil {
		*field = commandCfg[key].(T)
	}
}

func initBlockCommands(block *Block) {
	for _, command := range block.CommandsRaw {

		/* All this because for-vars can be a string referencing a list or list */
		Command := Command{}

		assignIfSet(command, "exec", &Command.Exec)
		assignIfSet(command, "diag", &Command.Diag)
		assignIfSet(command, "dir", &Command.Dir)
		assignIfSet(command, "condition", &Command.Condition)
		assignIfSet(command, "shell", &Command.Shell)
		assignIfSet(command, "shell_args", &Command.ShellArgs)

		checkSetDefault(&Command.Shell, block.Shell)
		checkSetDefault(&Command.ShellArgs, block.ShellArgs)

		if command["for-vars"] != nil {
			switch typeVal := command["for-vars"].(type) {
			/* inline list */
			case []any:

				for _, elem := range typeVal {
					Command.ForVars = append(Command.ForVars, elem.(string))
				}
			/* name of list */
			case string:

				if list := VarCfgs[typeVal]; list.ListValue != nil {
					Command.ForVars = list.ListValue
				}
			default:
				fmt.Printf("I don't know about type %T in for-vars!\n", typeVal)
			}
		} else {
			Command.ForVars = []string{"1"}
		}

		block.Commands = append(block.Commands, Command)
	}

	block.CommandsRaw = nil
}

type ExecConfig struct {
	Cmd    string
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
	Dir    string
}

func processBlock(block Block, config ExecConfig) {

	if len(block.Dir) > 0 {
		config.Dir = block.Dir
	} else {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot get current working directory \n")
			return
		} else {
			config.Dir = dir
		}
	}

	runCommandsWithConfig(block.Commands, config)
}

func runCommandsWithConfig(commands []Command, config ExecConfig) {

	cwd := config.Dir

	for _, command := range commands {

		if exit := checkCommandCondition(command.Condition, VarCfgs); exit != 0 {
			continue
		}

		execConfig := config

		/* Update cwd so that the directory update is
		   preserved until another command changes it */
		checkSetOverride(&cwd, render(command.Dir, VarCfgs))

		execConfig.Dir = cwd
		/* This behaves slightly different from the perl version
		   1. Diag wont override Exec and both can run if both are defined
		   2. Diag and Exec will both be looped with for-vars
		*/
		for index, value := range command.ForVars {

			varCfgs := map[string]VarCfg{}

			maps.Copy(varCfgs, VarCfgs)
			maps.Copy(varCfgs, map[string]VarCfg{"index": {StringValue: strconv.Itoa(index)}, "var": {StringValue: value}})

			if len(command.Diag) > 0 {
				execConfig.Cmd = "/usr/bin/echo"
				execConfig.Args = []string{render(command.Diag, varCfgs)}

				execCommand(execConfig)
			}

			if len(command.Exec) > 0 {
				execConfig.Cmd = command.Shell
				execConfig.Args = command.ShellArgs
				execConfig.Args = append(execConfig.Args, render(command.Exec, varCfgs))

				execCommand(execConfig)
			}
		}
	}
}

func execCommand(config ExecConfig) int {

	cmd := exec.Command(config.Cmd, config.Args...)
	cmd.Stdout = config.Stdout
	cmd.Stderr = config.Stderr
	cmd.Dir = config.Dir

	err := cmd.Run()
	if err != nil {

		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		} else {
			return 1
		}
	}

	return 0
}

func checkCommandCondition(condition string, varCfgs map[string]VarCfg) int {

	if len(condition) == 0 {
		return 0
	}

	config := ExecConfig{
		Stdout: os.NewFile(0, os.DevNull),
		Stderr: os.NewFile(0, os.DevNull),
	}

	config.Cmd = "/bin/bash"
	config.Args = []string{"-c", fmt.Sprintf("test %s", render(condition, varCfgs))}

	return execCommand(config)
}
