// Intention to use the dockerImage across stages
def dockerImage

pipeline {
    agent any

    environment {
        DOCKER_IMAGE = 'cloudsheeptech/shopping-list'
        DOCKER_TAG = 'latest'
        DOCKERFILE_NAME = 'Dockerfile.shopping-list-server'
        DATABASE = 'shopping_list_test'
    }

    tools {
        go '1.23'
        dockerTool 'docker-latest'
    }
    stages {
        // Since this is a pipeline, checkout of the repository happens automatically
        stage('Build') {
            steps {
                echo 'Building..'
                sh 'go build -o app-server ./cmd/shopping-list-server'
            }
        }
        stage('Test') {
            when {
                equals expected: "false", actual: "${env.SKIP_TESTS}"
            }
            steps {
                catchError(buildResult: 'UNSTABLE', stageResult: 'FAILURE') {
                    echo 'Testing..'
                    sh 'go test ./...'
                }
            }
        }
        stage('Prepare Docker Build') {
            steps {
                echo 'Copying important files'
                withCredentials([
                    file(credentialsId: 'shopping-list-db-secrets', variable: 'DB_SECRET_FILE'),
                    file(credentialsId: 'shopping-list-jwt-secret', variable: 'JWT_SECRET_FILE'),
                    file(credentialsId: 'shopping-list-certificate', variable: 'CERT_FILE'),
                    file(credentialsId: 'shopping-list-certificate-key', variable: 'KEY_FILE')
                ]) {
                    // Injecting credentials into the config files
                    sh 'mkdir -p resources'
                    sh 'cat "$DB_SECRET_FILE" > resources/dockerDb.json'
                    sh 'cat "$JWT_SECRET_FILE" > resources/jwtSecret.json'
                    sh 'cat "$CERT_FILE" > resources/shop.cloudsheeptech.com.crt'
                    sh 'cat "$KEY_FILE" > resources/shop.cloudsheeptech.com.pem'
                }
            }
        }
        stage('Docker Image') {
            steps {
                echo 'Building Docker Image'
                script {
                    dockerImage = docker.build("${env.DOCKER_IMAGE}:${env.DOCKER_TAG}", "-f ${env.DOCKERFILE_NAME} .")
                }
            }
        }
        stage('Deploy') {
            steps {
                echo 'Deploying docker build to dockerHub'
                script {
                    docker.withRegistry('https://index.docker.io/v1/', 'dockerhub-access-token') {
                        dockerImage.push()
                    }
                }
            }
        }
    }

    post {
        always {
            echo 'Cleanup secret files'
            sh 'rm -rf ./resources'
        }
        success {
            echo 'Pipeline Succeeded'
        }
        failure {
            echo 'Pipeline Failed'
        }
    }
}