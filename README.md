This script is designed to extract environment variables from a deployment, 
allowing you to replicate the runtime configuration on your local machine. 
It requires two input parameters: the context name of the target cluster and the deployment name.
The script will extract all environment variables, ConfigMap values, and Secret values, 
then save them to a .env file.

How to run this:
```shell
# Install dependencies 
# go mod download
go run cmd/main.go --context my-kubernetes-cluster-01 --deployment web-app-01
# Environment variables written to .env
```