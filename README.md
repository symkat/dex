# Dex - Directory Exec

**dex** is a tool to help you execute sets of commands in your project directories.

Similar in functionality to **make**, with a much simpler YAML file to define your commands in.  It supports nested commands and displays a menu of commands for your current directory.

**dex** is written in Go, so you just need to install a single binary with no dependencies.


## Installation

### Install binary from releases

1. Download the latest version [from the releases page](https://github.com/symkat/dex/releases).
2. Unpack and copy dex to /usr/local/bin, as shown below.

```
$ tar -xzf dex_*.tar.gz
$ sudo cp dex /usr/local/bin/dex
```

### Build & install from source
```
$ git clone https://github.com/symkat/dex.git
$ cd dex
$ go build -o dex main.go
$ sudo cp dex /usr/local/bin/dex
```

### User-only installation

Each of the above installation methods will make **dex** available for all users on the system, but requires root permission.

You can also copy **dex** into ~/bin and make sure that ~/bin is in your $PATH by adding `export PATH="$HOME/bin:$PATH"` to your `~/.bashrc` or `~/.bash_profile` file and restarting your terminal or running `source ~/.bashrc`.

## DexFile

The commands for your project directory are stored in a DexFile, **dex** will check for commands defined in `dex.yaml`, `dex.yml`, `.dex.yaml`, or `.dex.yml` in the current directory.  The first one of these files found is the one used.

The format of the dex file is:

```YAML
- name: build
  desc: This command will build the project
  shell:
    - first command to run
    - second command to run
```

This file would run `first command to run` and `second command to run` when invoked with `dex build`.

If you run `dex` it would show the menu

```
$ dex
build                   : This command will build the project
```

**dex** also supports nested commands.

Let's build a DexFile to support running ansible from our project directory to explore nested commands.

### DexFile for ansible

When initially developing a project and deploying it with Ansible, I'll use a file like this:

```YAML
- name: dev
  desc: "Commands on the development machine."
  children:
    - name: run-playbook
      desc: "Run the ansible playbook on the development machine."
      shell:
        -  ansible-playbook -i env/dev/inventory.yml --vault-password-file .vault_password -e @env/dev/vault.yml site.yml
    - name: edit-vault
      desc: "Edit the vault file."
      shell:
        - ansible-vault edit --vault-password-file .vault_password env/dev/vault.yml
    - name: encrypt-vault
      desc: "Encrypt the vault file."
      shell:
        - ansible-vault encrypt --vault-password-file .vault_password env/dev/vault.yml
    - name: decrypt-vault
      desc: "Decrypt the vault file."
      shell:
        - ansible-vault decrypt --vault-password-file .vault_password env/dev/vault.yml
- name: prod
  desc: "Manage the production cluster."
  children:
    - name: run-playbook
      desc: "Run the ansible playbook on the development machine."
      shell:
        -  ansible-playbook -i env/prod/inventory.yml --vault-password-file .vault_password -e @env/prod/vault.yml site.yml
    - name: edit-vault
      desc: "Edit the vault file."
      shell:
        - ansible-vault edit --vault-password-file .vault_password env/prod/vault.yml
```

Commands under **dev** work for my development environment, while commands under **prod** work for my production environment.

By codifying the commands in a DexFile I can use simple commands like `dex dev encrypt-vault` to encrypt my fault file, and `dex dev run-playbook` to install my project in my development environment.  When I'm ready to deploy to production, I can run `dex prod run-playbook`. 

```
$ dex
dev                     : Commands on the development machine.
    run-playbook            : Run the ansible playbook on the development machine.
    edit-vault              : Edit the vault file.
    encrypt-vault           : Encrypt the vault file.
    decrypt-vault           : Decrypt the vault file.
prod                    : Manage the production cluster.
    run-playbook            : Run the ansible playbook on the development machine.
    edit-vault              : Edit the vault file.
```
### Home Directory DexFile

You can keep a DexFile in your home directory to store global commands you might want to use outside a project directory. To use this DexFile just set `~~` as the first parameter in your command list.

### Config File Version 2

`dex` now has a new configuration format. The existing format is still supported and will function the same, but using this new format adds some new options and features that allow you to run more dynamic commands. 

```YAML
     version: 2
     vars:
       root_var: 'I can be used in every block'
       some_list:
         - 'this'
         - 'that'  
       work_dir: 
         from-command: pwd | tr -d '\n'  

     blocks:
       - name: var-example 
         desc: An Example block command with global and block variables.
         vars:
           some_string: 'for this block only' 
           env_var:
             from-env: SECOND_CMD 
             default: 0
         commands:
           - exec: echo 'Global var work_dir: [% work_dir %], block variable [% some_string %] '
           - exec: echo 'SECOND_CMD is set'
             condition: [%env_var%] -eq 1
       - name: loop-example
         desc: An Example block command that looks over a list var.
         commands: 
           - exec: echo 'repeating command with variable [% var %]
             for-vars: some_list  
```

The root `vars` attribute defines variables that can be used in any block by enclosing the name of the variable
within `[%` and `%]`.  These variables can be a string, number a list containing a combination of either. 

```YAML
     vars:
       string_var: 'I can be used in every block'
       number:var: 23423
       list_var:
         - 'foo'
         - 'bar'
         - 34
```

You can also configure variables to be initialized from the output of an external command or by referencing an environment variable.

```YAML
     vars:
       perl5_version: 
         from-command: "perl -MConfig -e 'print $Config{version}'"
         default: 'command failed'
       perl5_lib: 
         from-env: PERL5LIB
         default: 'NO PERL5LIB SET'

``` 

The `from-command` attribute will execute the set command and, assuming the command exits with a value of 0, assign its' STDOUT to the value of the variable. If the command returns multiple lines the variable will become a list containing
each line.  If the command exits with a non-zero value then the variable will be assigned the `default` attribute value
or remain undefined if no 'default' attribute is provided.

`from-env` will check for a matching environment variable and if found will assign that value to the variable. When the environment variable is not defined the 'default' attribute value is used.

`blocks` is similar to the root list in the Standard Format. It defines a list of named blocks of commands and nestable sub blocks of commands to run.  

```YAML
      blocks:
       - name: block-example
         desc: An Example block.
         vars:
           local_var: 'for this block only' 
         commands:
           - diag: '[%local_var%] execute update'
           - exec:  /bin/uptime
```

Within each block you can define `vars` with the same options the root `vars` attribute, but these variables will only be available for commands in that block.  

The `commands` attribute replaces the `shell` attribute and lets you define three kinds of commands.

  * `diag` - This command is an alias for echo and will print the string template to the terminal.

  * `dir`  - Sets the working directory for commands executed after this.  

  * `exec` - A command to execute.

The following configuration attributes are also available for each command.

  * `condition` - Takes a condition in the same format as the *test* command. If the condition returns false the command
    will be skipped.

  * `for-vars` - Can be a list or the name of variable that contains a list.  The command will be executed for each element of the list.  The value and index for each element in the list will be available as the `var` and `index` variables.

```YAML
      blocks:
       - name: for-vars-example
         desc: An Example block.
         vars:
           local_list: 
             - 1
             - 2
             - 3
         commands:
           - diag: 'value [%var%] at index [%index%]'
             for-vars: local_list
```     

## License

This software is copyright 2025 Kate Parkhurst and licensed under the MIT license.

## Availability

The latest version of this software can be found [in the GitHub repository](https://github.com/symkat/dex)


