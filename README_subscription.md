# Subscription - TO-DO
The subscrption mechanism has not been implemented in this release of the Discovery Agent.
The steps required for adding subscription support could look like the following:

1. Check if newly discovered API has an applied security policy
```
https://anypoint.mulesoft.com/apimanager/api/v1/organizations/{organizationId}/environments/{environmentId}/apis/{api-id}/policies
```
2. Check if API has SLA Tiers
```
https://anypoint.mulesoft.com/apimanager/api/v1/organizations/{organizationId}/environments/{environmentId}/apis/{environmentApiId}/tiers
```
3. Amplify CLI builds subscriptionDefinition using discovered SLA Tiers
```
…
plan:
   enum:
      - Gold
      - Silver
… 
```
4. Amplify CLI builds API deployment file.  Associates ConsumerInstance with matching subscriptionDefinition
5. When subsciption is approved, Amplify CLI uses discovered security policy to create an application and contract in Mulesoft API Gateway.
```
https://anypoint.mulesoft.com/exchange/api/v1/organizations/{organizationId}/applications

JSON body:
{
  "name" : "AppName"
}

https://anypoint.mulesoft.com/apimanager/api/v1/organizations/{organizationId}/environments/{environmentId}/apis/{environmentApiId}/contracts

JSON body:
{
    "applicationId": 3,
    "partyId": "",
    "partyName": "",
    "acceptedTerms": false,
    "requestedTierId": 48
}
```


Reference: [SDK Documentation - Building Discovery Agent](https://github.com/Axway/agent-sdk/blob/main/docs/discovery/index.md), [Mulesoft API Manager API](https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/api-manager-api/minor/1.0/console/method/%231156/)
