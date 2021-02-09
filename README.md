# agents-mulesoft

# Synopsis

This repository contains code fo the Mulesoft Anymount API Manager Discovery and Telemetry agents

# Discovery

The Discovery agent is used to identify APIs in the Mulesoft API Manager and publish them to the Amplify Catalog.

[in plan] If the API has security enabled then the discovery agent will create a subscription defenition which represents the available service tiers and create the required contract on the Mulesoft gateway.

# Telemetry

The telemetry agent queries the Mulesoft API gateway analytics interface and retrieves inormation about API calls and publishes it to the Amplify API Observer.