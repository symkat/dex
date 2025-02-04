# Dex - Directory Exec

**dex** is a tool to help you execute sets of commands in your project directories.

Similar in functionality to **make**, with a much simpler YAML file to define your commands in.  It supports nested commands and displays a menu of commands for your current directory.

**dex** is written in Go, so you just need to install a single binary with no dependencies.


## Installation

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

## License

This software is copyright 2025 Kate Parkhurst and licensed under the MIT license.

## Availability

The latest version of this software can be found [in the GitHub repository](https://github.com/symkat/dex)


