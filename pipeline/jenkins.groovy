pipeline {
    agent { label 'built-in' }
    
    parameters {
        choice(
            name: 'OS',
            choices: ['linux', 'darwin', 'windows'],
            description: 'Target operating system'
        )
        choice(
            name: 'ARCH',
            choices: ['amd64', 'arm64'],
            description: 'Target architecture'
        )
        booleanParam(
            name: 'SKIP_TESTS',
            defaultValue: false,
            description: 'Skip running tests'
        )
        booleanParam(
            name: 'SKIP_LINT',
            defaultValue: false,
            description: 'Skip running linter'
        )
        string(
            name: 'REGISTRY',
            defaultValue: 'ghcr.io/yurii-bielodied',
            description: 'GitHub Docker registry'
        )
        string(
            name: 'APP',
            defaultValue: 'kbot',
            description: 'Application name'
        )
        booleanParam(
            name: 'BUILD_IMAGE',
            defaultValue: true,
            description: 'Build Docker image (make image)?'
        )
        booleanParam(
            name: 'PUSH_IMAGE',
            defaultValue: false,
            description: 'Push Docker image to registry (make push)?'
        )
    }
    environment {
        TARGETARCH = "${params.TARGETARCH}"
        TARGETOS   = "${params.TARGETOS}"
        REGISTRY  = "${params.REGISTRY}"
        APP       = "${params.APP}"
    }

    options {
        buildDiscarder(logRotator(numToKeepStr: '5'))
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Set version') {
            steps {
                script {
                    def tag = sh(
                        script: 'git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0',
                        returnStdout: true
                    ).trim()
                    def shortSha = sh(
                        script: 'git rev-parse --short HEAD',
                        returnStdout: true
                    ).trim()

                    env.VERSION = "${tag}-${shortSha}"
                }
                echo "Build version: ${env.VERSION}"
            }
        }

        stage('Lint') {
            when {
                expression { !params.SKIP_LINT }
            }
            steps {
                sh '''
                    set -eu
                    make lint
                '''
            }
        }

        stage('Tests') {
            when {
                expression { !params.SKIP_TESTS }
            }
            steps {
                sh '''
                    set -eu
                    make test
                '''
            }
        }

        stage('Build binary') {
            steps {
                sh '''
                    set -eu
                    echo "Building for ${TARGETOS}/${TARGETARCH}"
                    make clean
                    make build
                '''
            }
        }

        stage('Docker image') {
            when {
                expression { return params.BUILD_IMAGE }
            }
            steps {
                sh '''
                    set -eu
                    echo "Building Docker image ${REGISTRY}/${APP}:${VERSION}-${TARGETOS}-${TARGETARCH}"
                    make image
                '''
            }
        }

        stage('Docker push') {
            when {
                expression { return params.BUILD_IMAGE && params.PUSH_IMAGE }
            }
            steps {
                sh '''
                    set -eu
                    echo "Pushing Docker image ${REGISTRY}/${APP}:${VERSION}-${TARGETOS}-${TARGETARCH}"
                    make push
                '''
            }
        }
    }

    post {
        success {
            echo "✅ Build OK: ${env.JOB_NAME} #${env.BUILD_NUMBER}"
        }
        failure {
            echo "❌ Build FAILED: ${env.JOB_NAME} #${env.BUILD_NUMBER}"
        }
        always {
            archiveArtifacts artifacts: 'kbot', fingerprint: true, onlyIfSuccessful: true
        }
    }
}
