#! groovy

pipeline {
	agent any
	stages {
		stage ('Build') {
			steps {
				sh '''
				brew instll operator-sdk
				export GO111MODULE=on
				go mod tidy
				operator-sdk build
				'''
			}
		}
	}
}