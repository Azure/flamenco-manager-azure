# Flamenco Deploy on Azure

Flamenco Manager + Workers can now be easily deployed on Microsoft Azure.

## Preparation

- Install [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli-apt?view=azure-cli-latest)
  and [Azure Batch Explorer](https://azure.github.io/BatchExplorer/).
- Run `az login` and log in via your browser.
- Make sure you have an SSH keypair available. The private key should be loaded into the SSH Agent
  (run `ssh-add -L` to check) or it should be an unencrypted key available in `$HOME/.ssh/id_rsa`.
  The public key is read from `$HOME/.ssh/id_rsa.pub`.


## Deploying Flamenco on Azure

First run the above preparation. The first time you run `flamenco-manager-azure` you may be asked the
following information:

  - **Subscription ID**: If you have a single Azure subscription, it's used automatically. If you
    have multiple, you'll have to choose which one to use.
  - **Physical location**: You'll get a list of locations to choose from.
  - **Resource Group**: All the resources (virtual machine, storage and batch accounts, virtual
    network components, etc.) created during the deployment will be contained in this group.
  - **Storage account name**: This name must be unique for the location of your choosing.
  - **Batch account name**: This name must be unique within the resource group.
  - **Virtual Machine name (for Flamenco Manager)**: This name is also used for the domain name
    assigned to the public IP address of the virtual machine, and as such must be unique for the
    location of your choosing.

After each prompt, your answer is stored in `azure_config.yaml`, and will be used in subsequent runs
of `flamenco-manager-azure`. If you want to change your answer, just delete the corresponding part of
`azure_config.yaml` and re-run `flamenco-manager-azure`.

The deployment takes approximately 10 minutes.


## After deployment

When deployment is done, Flamenco Manager is ready to be configured. The setup URL is logged at the
end of deployment, and will be `https://{VM name}.{location}.cloudapp.azure.com/setup`.

The Azure Batch pool can be resized using [Azure Batch Explorer](https://azure.github.io/BatchExplorer/).

To get the IP address of the virtual machine without re-running the deployment application, use:

    az network public-ip list --query [].ipAddress

## Blender Cloud Add-on configuration

The Blender Cloud Add-on should be configured to use the following settings:

- **Job Storage**: `shaman://{VM name}.{location}.cloudapp.azure.com/`. This is the same URL as the
  Manager, except replacing `https://` with `shaman://`.
- **Job Output**: `/mnt/flamenco-output/render`


## SSH Access

The Flamenco Manager VM can be reached via SSH using `ssh flamencoadmin@{VM name}.{location}.cloudapp.azure.com`.
The account's password is randomised and cannot be retrieved. Access is granted only using your private key.


## Get going with this Go code

Run:

    az login
    make
    ./flamenco-manager-azure -help

## Publishing a new release

Before creating a release, [create a Personal Access Token](https://github.com/settings/tokens)
with API access, then save it to the file `.GitHubAccessToken`.

Run:

    git tag v1.1  # replace with the release version
    git push --tags
    make release

If you get an error "Ref is not specified" from Gitlab, you aren't building exactly the same tag as
is available on Gitlab (so either your working directory is in a different Git revision, or you
didn't push the tag you're trying to build).


## Some more technical details

The [Azure Batch API Basics](https://docs.microsoft.com/en-us/azure/batch/batch-api-basics)
document is a nice place to start reading about Azure Batch. This document is also called
"Develop large-scale parallel compute solutions with Batch" and "Developer features".

- The files are in `/mnt/batch/tasks`:
    - `/mnt/batch/tasks/applications`: zipped and unzipped application packages.
      Note that these are suffixed with a datetime (I'm guessing node startup),
      so use the environment variables (see below) to refer to them.
    - `/mnt/batch/tasks/startup/std{out,err}.txt`: stdout and stderr output of
      the startup task.
    - `/mnt/batch/tasks/startup/wd`: default work directory of the startup task.
    - `/mnt/batch/tasks/shared`: this is *not* shared between VMs, but shared
      between tasks on that VM.
    - For more info see [Files and Directories](https://docs.microsoft.com/en-us/azure/batch/batch-api-basics#files-and-directories).
- `/var/lib/waagent` contains info from the Azure Agent, like the assigned
  hostname, configuration settings, TLS certificates, etc.
- If the pool is configured to run the startup task as `NonAdmin`, it uses
  `uid=1001(_azbatchtask_start) gid=1000(_azbatchgrp) groups=1000(_azbatchgrp)`.


## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
