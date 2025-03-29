package v2

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func check(t *testing.T, e error, s string) error {
	if e != nil {
		t.Errorf("%s - %v", s, e)
		return e
	}
	return nil
}

func createTestConfig(t *testing.T, config string) (*os.File, []byte, error) {

	data := []byte(config)

	tDexFile, err := os.CreateTemp("", "dex-test")
	check(t, err, "Error creating temp cfg file")

	_, err = tDexFile.Write(data)
	check(t, err, "Error writing to temp cfg file")

	yamlFile, err := os.Open(tDexFile.Name())
	check(t, err, "Error opening temp yaml file")

	yamlData, err := io.ReadAll(yamlFile)
	check(t, err, "Error reading yaml data")

	return tDexFile, yamlData, nil
}

func setupTestBlock(t *testing.T, test DexTest) (Block, *os.File, error) {

	tDexFile, yamlData, _ := createTestConfig(t, test.Config)

	dexFile, err := ParseConfig(yamlData)

	if err := check(t, err, "Error parsing config"); err != nil {
		return Block{}, nil, err
	}

	/* reset VarCfgs */
	VarCfgs = map[string]VarCfg{}

	initVars(dexFile.Vars)

	block, err := initBlockFromPath(dexFile, test.BlockPath)

	if err := check(t, err, "Error resolving command"); err != nil {
		return Block{}, nil, err
	}

	return block, tDexFile, nil
}

type DexTest struct {
	Name         string
	Config       string
	DexFile      DexFile2
	MenuOut      string
	BlockPath    []string
	Commands     []Command
	CommandsRaw  []map[string]any
	CommandOut   string
	ExpectedVars map[string]VarCfg
	Custom       func(t *testing.T, test DexTest, opts map[string]any)
}

func TestParseConfigFile(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Hello",
			Config: `---
version: 2
blocks:
  - name: hello
    desc: this is a command description`,
			DexFile: DexFile2{
				Version:   2,
				Shell:     "/bin/bash",
				ShellArgs: []string{"-c"},
				Blocks: []Block{
					{
						Name:        "hello",
						Desc:        "this is a command description",
						Commands:    nil,
						CommandsRaw: nil,
					},
				},
			},
		},
		{
			Name: "Hello Children",
			Config: `---
version: 2
shell: /bin/zsh
blocks:
  - name: hello
    desc: this is a command description
    children:
      - name: start
        desc: start the server
        commands:
          - exec: systemctl start server
      - name: stop
        desc: stop the server
        commands:
          - exec: systemctl stop server
            dir: /home/slice
      - name: restart
        desc: restart the server
        commands:
          - exec: systemctl stop server
          - exec: systemctl start server
`,
			DexFile: DexFile2{
				Version:   2,
				Shell:     "/bin/zsh",
				ShellArgs: []string{"-c"},
				Blocks: []Block{
					{
						Name: "hello",
						Desc: "this is a command description",
						Children: []Block{
							{
								Name:        "start",
								Desc:        "start the server",
								Commands:    nil,
								CommandsRaw: []map[string]any{{"exec": "systemctl start server"}},
							},
							{
								Name:        "stop",
								Desc:        "stop the server",
								Commands:    nil,
								CommandsRaw: []map[string]any{{"exec": "systemctl stop server", "dir": "/home/slice"}},
							},
							{
								Name:     "restart",
								Desc:     "restart the server",
								Commands: nil,
								CommandsRaw: []map[string]any{
									{"exec": "systemctl stop server"},
									{"exec": "systemctl start server"}},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {

		tcfg, yamlData, _ := createTestConfig(t, test.Config)

		defer os.Remove(tcfg.Name())

		dex_file, err := ParseConfig(yamlData)
		check(t, err, "config file not found")

		assert.Equal(t, test.DexFile, dex_file)

	}

}

func TestDisplayMenu(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Hello",
			Config: `---
version: 2
blocks:
  - name: hello
    desc: this is a command description`,
			MenuOut: "hello                   : this is a command description\n",
		},
		{
			Name: "Hello Children",
			Config: `---
version: 2
blocks:
  - name: hello
    desc: this is a command description
    children:
      - name: start
        desc: start the server
      - name: stop
        desc: stop the server
      - name: restart
        desc: restart the server
`,
			MenuOut: `hello                   : this is a command description
    start                   : start the server
    stop                    : stop the server
    restart                 : restart the server
`,
		},
	}

	for _, test := range tests {

		tcfg, yamlData, _ := createTestConfig(t, test.Config)

		defer os.Remove(tcfg.Name())

		dex_file, _ := ParseConfig(yamlData)

		var output bytes.Buffer
		displayMenu(&output, dex_file.Blocks, 0)

		assert.Equal(t, test.MenuOut, output.String())

	}

}

func TestResolveBlock(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Nested Command",
			Config: `---
version: 2
blocks:
  - name: server 
    desc: this is a command description
    children:
      - name: start
        desc: start the server
      - name: stop
        desc: stop the server
      - name: restart
        desc: restart the server
        commands: 
          - exec: restart server
          - exec: touch .restarted 
`,
			BlockPath: []string{"server", "restart"},
			CommandsRaw: []map[string]any{
				{"exec": "restart server"},
				{"exec": "touch .restarted"}},
		},
	}

	for _, test := range tests {

		tcfg, yamlData, _ := createTestConfig(t, test.Config)

		defer os.Remove(tcfg.Name())

		dex_file, err := ParseConfig(yamlData)

		check(t, err, "Error parsing config")

		block, err := resolveCmdToCodeblock(dex_file.Blocks, test.BlockPath)

		check(t, err, "Error resolving command")

		assert.Equal(t, test.CommandsRaw, block.CommandsRaw)

	}
}

func TestCommands(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Nested Command",
			Config: `---
version: 2
blocks:
  - name: hello_world 
    desc: this is a command description
    commands: 
       - exec: echo "hello world"
`,
			BlockPath:  []string{"hello_world"},
			CommandOut: "hello world\n",
		},
	}

	for _, test := range tests {

		block, tDexFile, err := setupTestBlock(t, test)

		defer os.Remove(tDexFile.Name())

		if err := check(t, err, "error setting up test"); err != nil {
			continue
		}

		var output bytes.Buffer

		config := ExecConfig{
			Stdout: &output,
			Stderr: &output,
		}

		processBlock(block, config)

		assert.Equal(t, test.CommandOut, output.String())

	}
}

func TestVars(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Global Vars",
			Config: `---
version: 2
vars: 
  string_var: "hi there"
  int_var: 2 
  list_var:
    - these
    - those
blocks:
  - name: hello_world 
    desc: this is a command description
    commands: 
       - exec: echo "hello world"
`,
			BlockPath:  []string{"hello_world"},
			CommandOut: "hello world\n",
			ExpectedVars: map[string]VarCfg{
				"string_var": {
					StringValue: "hi there",
				},
				"int_var": {
					StringValue: "2",
				},
				"list_var": {
					ListValue: []string{
						"these",
						"those",
					},
				},
			},
		},
		{
			Name: "Block Vars",
			Config: `---
version: 2
vars: 
  global_string: "foobar"

blocks:
  - name: block_vars 
    desc: this is a command description
    vars: 
      string_var: "from block"
      int_var: 3 
      list_var:
        - one 
        - two 
    commands: 
       - exec: echo "hello world"
  - name: other_block 
    desc: this is a command description
    vars: 
      string_var: "other local block var"

`,
			BlockPath:  []string{"block_vars"},
			CommandOut: "hello world\n",
			ExpectedVars: map[string]VarCfg{
				"global_string": {
					StringValue: "foobar",
				},
				"string_var": {
					StringValue: "from block",
				},
				"int_var": {
					StringValue: "3",
				},
				"list_var": {
					ListValue: []string{
						"one",
						"two",
					},
				},
			},
		},
		{
			Name: "Vars From Env",
			Config: `---
version: 2
vars:
  global_string:
    from-env: TESTENV 
  not_set:
    from_env: TESTENV_UNSET 
    default: fizzbizz 

blocks:
  - name: block_vars
    desc: this is a command description
`,
			BlockPath: []string{"block_vars"},
			ExpectedVars: map[string]VarCfg{
				"global_string": {
					FromEnv:     "TESTENV",
					StringValue: "from env!",
				},
				"not_set": {
					FromEnv:     "TESTENV_UNSET",
					Default:     "fizzbizz",
					StringValue: "fizzbizz",
				},
			},
		},
		{
			Name: "Vars From command",
			Config: `---
version: 2
vars:
  command_string:
    from-command: echo "c var" 
  command_list:
    from_command: echo -en "foo\nbar\nbazz" 

blocks:
  - name: block_vars
    desc: this is a command description
`,
			BlockPath: []string{"block_vars"},
			ExpectedVars: map[string]VarCfg{
				"command_string": {
					FromCommand: "echo \"c var\"",
					StringValue: "c var",
				},
				"command_list": {
					FromCommand: "echo -en \"foo\\nbar\\nbazz\"",
					ListValue:   []string{"foo", "bar", "bazz"},
				},
			},
		},
	}

	for _, test := range tests {

		os.Setenv("TESTENV", "from env!")

		_, tDexFile, err := setupTestBlock(t, test)

		defer os.Remove(tDexFile.Name())

		if err := check(t, err, "error setting up test"); err != nil {
			continue
		}

		assert.True(t, reflect.DeepEqual(test.ExpectedVars, VarCfgs))
	}
}

func TestRenderedCommand(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Global Vars",
			Config: `---
version: 2
vars: 
  string_var: "hi there"
blocks:
  - name: hello_world 
    desc: this is a command description
    commands: 
       - exec: echo "[%string_var%]"
`,
			BlockPath:  []string{"hello_world"},
			CommandOut: "hi there\n",
		},
		{
			Name: "Block Vars",
			Config: `---
version: 2
vars:
  global_string: "foobar"

blocks:
  - name: block_vars
    desc: this is a command description
    vars:
      string_var: "from block"
      int_var: 3
    commands:
       - exec: echo "[% global_string %] [%string_var%]-[%int_var%]"
`,
			BlockPath:  []string{"block_vars"},
			CommandOut: "foobar from block-3\n",
		},
		{
			Name: "Diag",
			Config: `---
version: 2
vars:
  global_string: "foobar"

blocks:
  - name: diag_command 
    desc: this is a command description
    vars:
      string_var: "from block"
      int_var: 4
    commands:
       - diag: "[% global_string %] [% string_var %] [% int_var %]"
`,
			BlockPath:  []string{"diag_command"},
			CommandOut: "foobar from block 4\n",
		},
	}

	for _, test := range tests {

		block, tDexFile, err := setupTestBlock(t, test)

		defer os.Remove(tDexFile.Name())

		if err := check(t, err, "error setting up test"); err != nil {
			continue
		}

		var output bytes.Buffer

		config := ExecConfig{
			Stdout: &output,
			Stderr: &output,
		}

		processBlock(block, config)

		assert.Equal(t, test.CommandOut, output.String())
	}
}

func TestCommandDir(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Block dir",
			Config: `---
version: 2
blocks:
  - name: change_dir 
    dir:  ".." 
    desc: this is a command description
    commands: 
       - exec: echo $(pwd)
`,
			BlockPath: []string{"change_dir"},
			Custom: func(t *testing.T, test DexTest, opts map[string]any) {

				output := opts["ouput"].(bytes.Buffer)

				path, _ := os.Getwd()

				parentDir := filepath.Dir(path) + "\n"

				assert.Equal(t, parentDir, output.String())
			},
		},
		{
			Name: "Command Dir",
			Config: `---
version: 2
blocks:
  - name: change_dir 
    desc: this is a command description
    vars:
      start_dir: 
        from-command: pwd
    commands: 
      - exec: echo $(pwd)
        dir:  ".." 
      - exec: echo $(pwd)
      - exec: echo $(pwd)
        dir:  "[% start_dir %]" 
      
`,
			BlockPath: []string{"change_dir"},
			Custom: func(t *testing.T, test DexTest, opts map[string]any) {

				output := opts["ouput"].(bytes.Buffer)

				path, _ := os.Getwd()
				newDir := filepath.Dir(path)

				parentDir := newDir + "\n" + newDir + "\n" + path + "\n"

				assert.Equal(t, parentDir, output.String())
			},
		},
	}

	for _, test := range tests {

		block, tDexFile, err := setupTestBlock(t, test)

		defer os.Remove(tDexFile.Name())

		if err := check(t, err, "error setting up test"); err != nil {
			continue
		}

		var output bytes.Buffer

		config := ExecConfig{
			Stdout: &output,
			Stderr: &output,
		}

		processBlock(block, config)

		test.Custom(t, test, map[string]any{"ouput": output})
	}
}

func TestForVars(t *testing.T) {

	tests := []DexTest{
		{
			Name: "for-vars",
			Config: `---
version: 2
blocks:
  - name: loop_vars 
    dir:  ".." 
    desc: this is a command description
    commands: 
      - exec: echo [% index %] [% var %] 
        for-vars: 
          - one
          - two
          - three
`,
			BlockPath:  []string{"loop_vars"},
			CommandOut: "0 one\n1 two\n2 three\n",
		},
		{
			Name: "for-vars list ref",
			Config: `---
version: 2
vars:
  some_string: foobar
  some_list:
    - four
    - five
    - six
blocks:
  - name: loop_vars 
    dir:  ".." 
    desc: this is a command description
    commands: 
      - exec: echo [% some_string %] [% index %] [% var %] 
        for-vars: some_list 
`,
			BlockPath:  []string{"loop_vars"},
			CommandOut: "foobar 0 four\nfoobar 1 five\nfoobar 2 six\n",
		},
	}

	for _, test := range tests {

		block, tDexFile, err := setupTestBlock(t, test)

		defer os.Remove(tDexFile.Name())

		if err := check(t, err, "error setting up test"); err != nil {
			continue
		}

		var output bytes.Buffer

		config := ExecConfig{
			Stdout: &output,
			Stderr: &output,
			Dir:    block.Dir,
		}

		processBlock(block, config)

		assert.Equal(t, test.CommandOut, output.String())
	}
}

func TestCommandCondition(t *testing.T) {

	tests := []DexTest{
		{
			Name: "conditions",
			Config: `---
version: 2
blocks:
  - name: condition commands 
    dir:  ".." 
    desc: this is a command description
    commands: 
      - exec: echo condition true 
        condition: 1 -eq 1 
      - exec: echo condition false 
        condition: 1 -eq 0 

`,
			BlockPath:  []string{"condition commands"},
			CommandOut: "condition true\n",
		},
		{
			Name: "conditions",
			Config: `---
version: 2
vars:
  conditionVal: 1
blocks:
  - name: condition commands 
    dir:  ".." 
    desc: this is a command description
    commands: 
      - exec: echo condition true 
        condition: 1 -eq [% conditionVal %] 
`,
			BlockPath:  []string{"condition commands"},
			CommandOut: "condition true\n",
		},
	}

	for _, test := range tests {

		block, tDexFile, err := setupTestBlock(t, test)

		defer os.Remove(tDexFile.Name())

		if err := check(t, err, "error setting up test"); err != nil {
			continue
		}

		var output bytes.Buffer

		config := ExecConfig{
			Stdout: &output,
			Stderr: &output,
		}

		processBlock(block, config)

		assert.Equal(t, test.CommandOut, output.String())
	}
}
