//go:build zsh

package v2

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZshShellCommand(t *testing.T) {

	tests := []DexTest{
		{
			Name: "zshell",
			Config: `---
version: 2
shell: /usr/bin/zsh
blocks:
  - name: zsh command 
    dir:  ".." 
    desc: this is a command description
    commands: 
      - exec: echo from $0! 
`,
			BlockPath:  []string{"zsh command"},
			CommandOut: "from /usr/bin/zsh!\n",
		},
		{
			Name: "zshell",
			Config: `---
version: 2
blocks:
  - name: block zsh command 
    shell: /usr/bin/zsh
    dir:  ".." 
    desc: this is a command description
    commands: 
      - exec: echo from $0! 
`,
			BlockPath:  []string{"block zsh command"},
			CommandOut: "from /usr/bin/zsh!\n",
		},
		{
			Name: "zshell",
			Config: `---
version: 2
blocks:
  - name: zsh for one command 
    dir:  ".." 
    desc: this is a command description
    commands: 
      - exec: echo from $0! 
        shell: /usr/bin/zsh
      - exec: echo from $0! 
`,
			BlockPath:  []string{"zsh for one command"},
			CommandOut: "from /usr/bin/zsh!\nfrom /bin/bash!\n",
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
