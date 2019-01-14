# Azure+Go test

## Get going

- Install the [Azure CLI client](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli-apt?view=azure-cli-latest).
- Run `az login` and log in via your browser. Later you can run `az account list` to show the same info.
  The fields in the shown JSON map as follows:

    - `id` = subscription ID
    - `tenantId` = Tentant ID

- Run `az ad sp create-for-rbac --sdk-auth > client_credentials.json`
- Run `export AZURE_AUTH_LOCATION=$(pwd)/client_credentials.json`

- Install [dep](https://github.com/golang/dep#installation)
- Run `dep ensure`


To run this example, run `go install` and then `azure-go-test -debug`.
