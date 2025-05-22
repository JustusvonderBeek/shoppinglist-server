pipeline {
    agent any

    environment {
        DOCKER_IMAGE = 'cloudsheeptech/shopping-list-server'
        DOCKER_TAG = 'latest'
        DOCKERFILE_NAME = 'Dockerfile.multistage'
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
//         stage('Test') {
//             steps {
//                 echo 'Testing..'
//                 sh 'go test ./...'
//             }
//         }
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
                script {
                    docker.withRegistry('https://hub.docker.com/r', 'dockerhub-access-token') {
                        dockerImage.push()
                    }
                }
            }
        }
    }

    post {
        success {
            echo 'Pipeline Succeeded'
        }
        failure {
            echo 'Pipeline Failed'
        }
    }
}