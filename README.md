# Amplify MuleSoft AnyPoint Agent

## Overview

This repository contains the Axway Amplify dataplane agents for the MuleSoft AnyPoint platform. These agents connect your MuleSoft Anypoint dataplane to the Amplify Central management plane.

## Prerequisites

* You need an Axway Platform user account that is assigned the AMPLIFY Central admin role
* Your MuleSoft AnyPoint deployment should be up and running and have APIs to be discovered and exposed in AMPLIFY Central

Let’s get started!

## Prepare AMPLIFY Central Environments

In this section we'll:

* [Create an environment in Central](#create-an-environment-in-central)
* [Create a service account](#create-a-service-account)

### Create an environment in Central

* Log into [Amplify Central](https://apicentral.axway.com)
* Navigate to "Topology" then "Environments"
* Click "+ Environment"
  * Select a name
  * Click "Save"
* To enable the viewing of the agent status in Amplify see [Visualize the agent status](https://docs.axway.com/bundle/amplify-central/page/docs/connect_manage_environ/environment_agent_resources/index.html#add-your-agent-resources-to-the-environment)

### Create a service account

* Create a public and private key pair locally using the openssl command

```sh
openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits: 2048
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

* Log into the [Amplify Platform](https://platform.axway.com)
* Navigate to "Organization" then "Service Accounts"
* Click "+ Service Account"
  * Select a name
  * Optionally add a description
  * Select "Client Certificate"
  * Select "Provide public key"
  * Select or paste the contents of the public_key.pem file
  * Select "Central admin"
  * Click "Save"
* Note the Client ID value, this and the key files will be needed for the agents

## Prepare MuleSoft

* Gather the following information
  * MuleSoft AnyPoint Exchange URL
  * MuleSoft Environment name to discover and track transactions from (e.g. Sandbox)
  * MuleSoft AnyPoint Business Unit the agent connects to
  * MuleSoft Username and Password or Client ID and Secret.
    * If using a Client ID and Secret then a MuleSoft App Integration will need to be created

## Setup agent Environment Variables

The following environment variables file should be created for executing both of the agents

```ini
CENTRAL_ORGANIZATIONID=<Amplify Central Organization ID>
CENTRAL_TEAM=<Amplify Central Team Name>
CENTRAL_ENVIRONMENT=<Amplify Central Environment Name>   # created in Prepare AMPLIFY Central Environments step

CENTRAL_AUTH_CLIENTID=<Amplify Central Service Account>  # created in Prepare AMPLIFY Central Environments step
CENTRAL_AUTH_PRIVATEKEY=/keys/private_key.pem            # path to the key file created with openssl
CENTRAL_AUTH_PUBLICKEY=/keys/public_key.pem              # path to the key file created with openssl

MULESOFT_ANYPOINTEXCHANGEURL=https://mulesoftexhange.com # gathered in Prepare MuleSoft step
MULESOFT_AUTH_USERNAME=username                          # gathered in Prepare MuleSoft step
MULESOFT_AUTH_PASSWORD=password                          # gathered in Prepare MuleSoft step
MULESOFT_ENVIRONMENT=Sandbox                             # gathered in Prepare MuleSoft step
MULESOFT_ORGNAME=Unit                                    # gathered in Prepare MuleSoft step

LOG_LEVEL=info
LOG_OUTPUT=stdout
```

## Discovery Agent

Reference: [Discovery Agent](/README_discovery.md)

## Traceability Agent

Reference: [Traceability Agent](/README_traceability.md)

## Development and Compiling

The MuleSoft AnyPoint agent repository delivers only Docker containers, if it is desired to run an agent outside of a Docker container the agents may be compiled for a specific OS (`GOOS`) and architecture (`GOARCH`).

Agents running outside of the delivered docker containers are considered to be in a development state and do not receive the same level of support as the delivered Docker containers.

### Building

1. Install golang, a version the same or newer of that defined in the `go.mod` file
2. Fork/Clone this git repository locally `git clone git@github.com:Axway/agents-mulesoft.git`
3. Run the following commands changing the OS and Architecture as necessary

Discovery

```bash
CGO_ENABLED=0  GOOS=linux GOARCH=amd64  make build-discovery
```

Traceability

```bash
CGO_ENABLED=0  GOOS=linux GOARCH=amd64  make build-traceability
```

The golang website may be referenced for supported values in those variables [Go Docs](https://pkg.go.dev/internal/platform).
