# agents-mulesoft

This repo maintains the software allowing for an Axway Amplify agent to proxy dataplane api discovery, subscription and telemetry from Mulesoft Anypoint Platform to Axway Amplify  

## Prerequisites

* A go friendly environment with make 

# Getting started

run `make`

## Create an environment in Central

Log into Amplify Central https://apicentral.axway.com
Navigate to the Topology page
Click the "Environment" button in the top right.
Select "Other" for the gateway type.
Provide a name and a title, such as "mulesoft-gateway" and then hit "Save" in the top right.

## Create a DOSA Account

Create a public and private key pair locally on your computer.
```shell
openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits:2048
openssl rsa -in private_key.pem -pubout -out public_key.pem
```
In Central, click the "Access" tab on the sidebar, which is the second to last tab.
Click on "Service Accounts".
Click the button in the top right that says "+ Service Account".
Name the account and provide the public key.

## Find your Organization ID

After making the environment click on your name in the top right. Select "Organization" from the dropdown.
You will see a field called "Organization ID". This will be needed to connect the agents to your org.

## Anypoint Platform

//TODO

## Fill out the environment variables

// TODO 

# mulesoft Discovery Agent

//TODO

# mulesoft Traceability Agent

// TODO

## Build and run the binary

//TODO

# Development

//TODO