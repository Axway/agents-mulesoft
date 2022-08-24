#!/bin/bash

sonar-scanner -X \
    -Dsonar.host.url=${SONAR_HOST} \
    -Dsonar.language=go \
    -Dsonar.projectName=Mulesoft_Agents \
    -Dsonar.projectVersion=1.0 \
    -Dsonar.projectKey=Mulesoft_Agents \
    -Dsonar.sourceEncoding=UTF-8 \
    -Dsonar.projectBaseDir=${WORKSPACE} \
    -Dsonar.sources=. \
    -Dsonar.tests=. \
	-Dsonar.exclusions=**/*.json \
    -Dsonar.test.inclusions=**/*test*.go \
    -Dsonar.go.tests.reportPaths=goreport.json \
    -Dsonar.go.coverage.reportPaths=gocoverage.out \
    -Dsonar.issuesReport.console.enable=true \
    -Dsonar.report.export.path=sonar-report.json \
    -Dsonar.verbose=true
