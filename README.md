# Amplify Mulesoft Anypoint Agent

## Overview

This repository contains the Axway Amplify dataplane agents for the Mulesoft Anypoint platform. These agents connect your Mulesoft Anypoint dataplane to the Amplify Central management plane.

### Discovery

The Discovery agent is used to discover APIs in the Mulesoft API Manager and publish them to the Amplify Central.

### Traceability

The traceability agent retrieves traffic and usage data from the Mulesoft API Manager analytics service and publishes it to the Amplify Central.

## Getting Started

The setup process is:
- Download and install the agents.
- Create a Service Account in Amplify Central for the agents to use.
- Create an Environment in Amplify Central for the agents to publish too.
- Configure and run the agents.

### Downloading the Agents
Download and unzip the [latest release](https://github.com/Axway/agents-mulesoft/releases/latest) of the *discovery agent* and the *traceability agent*.

```
curl  -L https://github.com/Axway/agents-mulesoft/releases/download/v1.0.0/mulesoft_discovery_agent_1.0.0_Linux_x86_64.tar.gz --output - | tar xz
curl  -L https://github.com/Axway/agents-mulesoft/releases/download/v1.0.0/mulesoft_traceability_agent_1.0.0_Linux_x86_64.tar.gz --output - | tar xz
```

### Configure Axway Amplify Central
Navigate to [https://platform.axway.com](https://platform.axway.com) and authenticate or sign up for a trial account.

#### Locate Amplify Organization ID

<img src="./img/WelcomeToAmplify.png" width="600">

Click on your profile in the top-right corner of the Welcome screen and select *Organization*.

<img src="./img/OrganizationID.png" width="600">

Note the value of the Organization ID.

#### Create a Service Account
Service Account are used by Amplify so that the Agents can connect securely to Amplify Central using private key credentials known only to the owner of the dataplane.

##### Using the Axway CLI
The creation of the service account requires a public/private key pair. The Axway CLI can automatically generate these and create the service account.

```
$ amplify central create service-account
WARNING: Creating a new DOSA account will overwrite any existing "private_key.pem" and "public_key.pem" files in this directory
? Enter a new service account name:  ExampleSA
Creating a new service account.
New service account "ExampleSA" with clientId "DOSA_edf194aa2430422bace013ce46a31d4a" has been successfully created.
The private key has been placed at /home/user/example/private_key.pem
The public key has been placed at /home/user/example/public_key.pem
```

For more information on configuring the Axway CLI see [Getting started with Amplify Central CLI](https://docs.axway.com/bundle/axway-open-docs/page/docs/central/cli_central/index.html).

##### Using the Amplify Central UI
Click the grid icon at the top-left of the UI and select *Central*.

Navigate to *Access -> Service Accounts*.

Click the `+Service Account` Button.

Add a name and a public key.

To generate a public key, you can install OpenSSL and run the commands:

```
openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits:2048
openssl rsa -pubout -in private_key.pem -out public_key.pem
```

<img src="./img/ServiceAccount.png" width="600">

Note the Client ID value.

### Create Environment
The environment in Amplify Central is where the agent will publish the resources it discovers from Mulesoft Anypoint API Manager.

Navigate to the Toplogy tab and click the `+Environment` Button

Complete the configuration form, noting the value entered in the `Name` field. It must be all lowercase with no spaces as it will be used as an identifier to the agent configuration later.

<img src="./img/Environments.png" width="600">

### Agent Configuration
To configure the agents see:
- [Mulesoft Discovery Agent Configuration](./README_discovery.md)
- [Mulesoft Traceability Agent Configuration](./README_traceability.md)

### Agent Status Reporting
In order for the  environment status in Amplify Central / Topology to display the connected agent status, we have to create the necessary resources for the Agents. 
Please note that you would need the Axway CLI to create these resources.
Given below are the sample configuration for the agents:

Discovery Agent:
```
group: management
apiVersion: v1alpha1
kind: DiscoveryAgent
name: my-agent-name
title: My DiscoveryAgent title
metadata:
  scope:
    kind: Environment
    name: my-amplify-central-environment
attributes: {}
finalizers: []
tags:
  - sample
spec:
  config:
    additionalTags:
      - DiscoveredByMulesoftAgent
  logging:
    level: debug
  dataplaneType: my-dataplane-name
```
Traceability Agent:
```
group: management
apiVersion: v1alpha1
kind: TraceabilityAgent
name: my-agent-name
title: My beautiful TraceabilityAgent title
metadata:
  scope:
    kind: Environment
    name: my-amplify-central-environment
attributes: {}
finalizers: []
tags:
  - sample
spec:
  config:
    excludeHeaders:
      - Authorization
    processHeaders: true
  dataplaneType: my-dataplane-name
```
Once configured, save those files and please create those resources:
```
axway central apply -f disovery-agent-res.yaml
---
axway central apply -f traceability-agent-res.yaml
```
In order to link agent deployment with the appropriate agent resource, 
you have to update the agent configuration file (env_vars). Use the `CENTRAL_AGENTNAME` variable and link the value to the resource name defined previously.

Once the Agents successfully starts, the agent status (AMPLIFY Central / Topology) will change to `Running`. If there are no other agents linked to that environment, then the environment status will change from `Manual Sync` to `Connected`.
So in the traceability-deployment.yaml and discovery-deployment.yaml, set the Env Variable:
```
- name: CENTRAL_AGENTNAME
  value: my-agent-name
```
For more information on setting up the resources for Visualizing Agent Status reporting, please refer to:  [Visualizing Agent Resources](https://docs.axway.com/bundle/axway-open-docs/page/docs/central/env_gw_mgmt/environment_agent_resources/index.html)

## See Also
Reference: [SDK Documentation - Building Discovery Agent](https://github.com/Axway/agent-sdk/blob/main/docs/discovery/index.md), [Mulesoft API Manager API](https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/api-manager-api/minor/1.0/console/method/%231156/)
