# Amplify Mulesoft Anypoint Traceability Agent

## Prerequisites
Install the agent and provision Amplify Central access as described in [https://github.com/Axway/agents-mulesoft/blob/main/README.md](https://github.com/Axway/agents-mulesoft/blob/main/README.md).

- Amplify organization id
- Amplify Central environment name
- Pubic/Private key pem files
- Service account client id

As well as access to Amplify Central it is assumed you have access to the [Mulesoft Anypoint Platform](https://anypoint.mulesoft.com/exchange). You need:

- Credentials with access to the organization the agents will attach to.
- Mulesoft environment name to discover from (e.g. Sandbox)


## Configuring the Traceability Agent

The agents read their configuration from a YAML file. To set up your config file copy the content of `default_mulesoft_traceability_agent.yml` into a new file named `mulesoft_traceability_agent`, and replace the default values that reflect your environment.

The minimial configuration for the agent is:

```
central:
  organizationID: <your organization id>
  environment: <your amplify environment name>
  auth:
    clientID: <your service account client id>
    privateKey: <path to your private_key.pem>
    publicKey: <path to your public_key.pem>

mulesoft:
 environment: <your Mulesoft environment>
 auth:
  username: <your Mulesoft username>
  password: <your Mulesoft password>
```

## Start the Traceability Agent

```
./mulesoft_traceability_agent --pathConfig <path to mulesoft_traceability_agent.yaml>
```

## Configuration Variables

- The following are all of the Environment variables that can be set, they will override the defaults

| Variable Name                  |  YAML Path                                                           |Description                                                                                                                                                                                                                                                                                              | **Location** / _Default_                                             |
| ------------------------------ | ---------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| CENTRAL_AUTH_CLIENTID          | central.auth.clientId                                                |The DOSA ID of the AMPLIFY Central Service Account created                                                                                                                                                                                                                                               | **AMPLIFY Central -> Access -> Service Accounts**                    |
| CENTRAL_AUTH_KEYPASSWORD       | central.auth.keyPassword                                             |The password for the private key, if applicable                                                                                                                                                                                                                                                          |                                                                      |
| CENTRAL_AUTH_PRIVATEKEY        | central.auth.privateKey                                              |The private key file path from the commands above                                                                                                                                                                                                                                                        | _/keys/private_key.pem_                                              |
| CENTRAL_AUTH_PUBLICKEY         | central.auth.publicKey                                               |The public key file path from the commands above                                                                                                                                                                                                                                                         | _/keys/public_key.pem_                                               |
| CENTRAL_AUTH_REALM             | central.auth.realm                                                   |The Realm used to authenticate for AMPLIFY Central                                                                                                                                                                                                                                                       | _Broker_                                                             |
| CENTRAL_AUTH_URL               | central.auth.url                                                     |The URL used to authenticate for AMPLIFY Central                                                                                                                                                                                                                                                         | _https://login.axway.com/auth_                                       |
| CENTRAL_DEPLOYMENT             | central.deployment                                                   |The AMPLIFY Central deployment environment (beano, dev, prod, preprod)                                                                                                                                                                                                                                   | _prod_                                                               |
| CENTRAL_ENVIRONMENT            | central.environment                                                  |The Environment Name for the AMPLIFY Central Environment                                                                                                                                                                                                                                                 | **See Instructions below**                                           |
| CENTRAL_ORGANIZATIONID         | central.platformURL                                                  |The Organization ID from AMPLIFY Central                                                                                                                                                                                                                                                                 | **Platform -> Click User -> Organization**                           |
| CENTRAL_SSL_CIPHERSUITES       | central.ssl.cipherSuites                                             |An array of strings. It is a list of supported cipher suites for TLS versions up to TLS 1.2. If CipherSuites is nil, a default list of secure cipher suites is used, with a preference order based on hardware performance. [See below](#supported-cipher-suites) for currently supported cipher suites. | [See below](#default-cipher-suites) for default cipher suite setting |
| CENTRAL_SSL_INSECURESKIPVERIFY | central.ssl.insecureSkipVerify                                       |InsecureSkipVerify controls whether a client verifies the server's certificate chain and host name. If InsecureSkipVerify is true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.             | Internally defaulted to false                                        |
| CENTRAL_SSL_MAXVERSION         | central.ssl.maxVersion                                               |String value for the maximum SSL/TLS version that is acceptable. If empty, then the maximum version supported by this package is used, which is currently TLS 1.3. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3                                                                                    | Internally, this value defaults to empty                             |
| CENTRAL_SSL_MINVERSION         | central.ssl.minVersion                                               |String value for the minimum SSL/TLS version that is acceptable. If zero, empty TLS 1.0 is taken as the minimum. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3                                                                                                                                      | Internally, the value defaults toTLS1.2                              |
| CENTRAL_SSL_NEXTPROTOS         | central.ssl.nextProtos                                               |An array of strings. It is a list of supported application level protocols, in order of preference, based on the ALPN protocol list. Allowed values are: h2, htp/1.0, http/1.1, h2c                                                                                                                      | Internally empty. Default negotiation.                               |
| CENTRAL_URL                    | central.URL                                                          |The URL to the AMPLIFY Central instance being used for this traceability agent                                                                                                                                                                                                                           | _https://apicentral.axway.com_                                       |
| LOG_FORMAT                     | log.format                                                           |The format to print log messages (json, line, package)                                                                                                                                                                                                                                                   | _json_                                                               |
| LOG_LEVEL                      | log.level                                                            |The log level for output messages (debug, info, warn, error)                                                                                                                                                                                                                                             | _info_                                                               |
| LOG_OUTPUT                     | log.output                                                           |The output for the log lines (stdout, file, both)                                                                                                                                                                                                                                                        | _stdout_                                                             |
| LOG_PATH                       | log.path                                                             |The path (relative or absolute) to save logs files, if output type file or both                                                                                                                                                                                                                          | _logs_                                                               |
| MULESOFT_ANYPOINTEXCHANGEURL   | mulesoft.anypointExchangeUrl                                        | Mulesoft Anypoint Exchange URL                                                                                                                                                                                                                                                                           | https://anypoint.mulesoft.com                                        |
| MULESOFT_AUTH_LIFETIME         | mulesoft.auth.lifetime                                              | The session lifetime. The agent will automatically refresh the access token as it approaches the end of its lifetime                                                                                                                                                                                     | 60m                                                                  |
| MULESOFT_AUTH_PASSWORD         | mulesoft.auth.password                                              | The password for the Mulesoft Anypoint username created for this agent                                                                                                                                                                                                                                   |                                                                      |
| MULESOFT_AUTH_USERNAME         | mulesoft.auth.username                                              | The Mulesoft Anypoint username created for this agent                                                                                                                                                                                                                                                    |                                                                      |
| MULESOFT_CACHEPATH             | mulesoft.cachePath                                                  | Path entry to store stateful cache between agent invocations                                                                                                                                                                                                                                             | _/tmp_                                                               |
| MULESOFT_ENVIRONMENT           | mulesoft.environment                                                | The Mulesoft Anypoint Exchange the agent connects to, e.g. Sandbox.                                                                                                                                                                                                                                      |                                                                      |
| MULESOFT_POLLINTERVAL          | mulesoft.pollInterval                                               | The frequency in which Mulesoft API Manager is polled for new endpoints.                                                                                                                                                                                                                                 | _30s_                                                                |
| MULESOFT_PROXYURL              | mulesoft.proxyUrl                                                   | The url for the proxy for API Manager (e.g. http://username:password@hostname:port). If empty, no proxy is defined.                                                                                                                                                                                      | Internally, this value defaults to empty                             |
| MULESOFT_SSL_CIPHERSUITES      | mulesoft.ssl.cipherSuites                                           | An array of strings. It is a list of supported cipher suites for TLS versions up to TLS 1.2. If CipherSuites is nil, a default list of secure cipher suites is used, with a preference order based on hardware performance. [See below](#supported-cipher-suites) for currently supported cipher suites. | [See below](#default-cipher-suites) for default cipher suite setting |
| MULESOFT_SSL_INSECURESKIPVERIFY | mulesoft.ssl.insecureSkipVerify                                    | InsecureSkipVerify controls whether a client verifies the server's certificate chain and host name. If InsecureSkipVerify is true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.             | Internally defaulted to false                                        |
| MULESOFT_SSL_MAXVERSION        | mulesoft.ssl.maxVersion                                             | String value for the maximum SSL/TLS version that is acceptable. If empty, then the maximum version supported by this package is used, which is currently TLS 1.3. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3                                                                                    | Internally, this value defaults to empty                             |
| MULESOFT_SSL_MINVERSION        | mulesoft.ssl.minVersion                                             | String value for the minimum SSL/TLS version that is acceptable. If zero, empty TLS 1.0 is taken as the minimum. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3                                                                                                                                      | Internally, the value defaults toTLS1.2                              |
| MULESOFT_SSL_NEXTPROTOS        | mulesoft.ssl.nestProtos                                             | An array of strings. It is a list of supported application level protocols, in order of preference, based on the ALPN protocol list. Allowed values are: h2, htp/1.0, http/1.1, h2c                                                                                                                      | Internally empty. Default negotiation.                               |
| STATUS_HEALTHCHECKINTERVAL     | sstatus.healthCheckInterval                                          |Time in seconds between running periodic health checker (binary agents only). Allowed values are from 30 to 300 seconds.                                                                                                                                                                                      | _30s_                                                                 |
| STATUS_HEALTHCHECKPERIOD       | status.healthCheckPeriod                                             |Time in minutes allotted for services to be ready before exiting the agent. Allowed values are from 1 to 5 minutes.                                                                                                                                                                                      | _3m_                                                                 |
| STATUS_PORT                    | status.port                                                          |The port that the healthcheck endpoint will listen on                                                                                                                                                                                                                                                    | _8989_                                                               |
| TRACEABILITY_COMPRESSIONLEVEL  | output.traceability.compression_level                                |The gzip compression level for the output event. Setting this to 0 will disable the compression                                                                                                                                                                                                          | Defaults to _3_                                                      |
| TRACEABILITY_HOST              | output.traceability.host                                            |Host name and port of the ingestion service to forward the transaction log entries,                                                                                                                                                                                                                      | _ingestion-lumberjack.datasearch.axway.com:453_                      |
| TRACEABILITY_PROTOCOL          | output.traceability.protocol                                         |Protocol(https or tcp) to be used for communicating with ingestion service                                                                                                                                                                                                                               | tcp                                                                  |
| TRACEABILITY_PROXYURL          | output.traceability.proxy_url                                        |The url for the proxy for ingestion service (e.g. socks5://hostname:port). If empty, no proxy is defined.                                                                                                                                                                                                | Internally, this value defaults to empty                             |


### Supported Cipher Suites

The allowed cipher suites string values are allowed: ECDHE-ECDSA-AES-128-CBC-SHA, ECDHE-ECDSA-AES-128-CBC-SHA256, ECDHE-ECDSA-AES-128-GCM-SHA256, ECDHE-ECDSA-AES-256-CBC-SHA, ECDHE-ECDSA-AES-256-GCM-SHA384, ECDHE-ECDSA-CHACHA20-POLY1305, ECDHE-ECDSA-RC4-128-SHA, ECDHE-RSA-3DES-CBC3-SHA, ECDHE-RSA-AES-128-CBC-SHA, ECDHE-RSA-AES-128-CBC-SHA256, ECDHE-RSA-AES-128-GCM-SHA256, ECDHE-RSA-AES-256-CBC-SHA, ECDHE-RSA-AES-256-GCM-SHA384, ECDHE-RSA-CHACHA20-POLY1305, ECDHE-RSA-RC4-128-SHA, RSA-RC4-128-SHA, RSA-3DES-CBC3-SHA, RSA-AES-128-CBC-SHA, RSA-AES-128-CBC-SHA256, RSA-AES-128-GCM-SHA256, RSA-AES-256-CBC-SHA, RSA-AES-256-GCM-SHA384, TLS-AES-128-GCM-SHA256, TLS-AES-256-GCM-SHA384, TLS-CHACHA20-POLY1305-SHA256

### Default Cipher Suites

The list of default cipher suites is: ECDHE-ECDSA-AES-256-GCM-SHA384, ECDHE-RSA-AES-256-GCM-SHA384, ECDHE-ECDSA-CHACHA20-POLY1305, ECDHE-RSA-CHACHA20-POLY1305, ECDHE-ECDSA-AES-128-GCM-SHA256, ECDHE-RSA-AES-128-GCM-SHA256, ECDHE-ECDSA-AES-128-CBC-SHA256, ECDHE-RSA-AES-128-CBC-SHA256