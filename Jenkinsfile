pipeline {
    agent any

    environment {
        DOCKER_IMAGE = 'cloudsheeptech/shopping-list'
        DOCKER_TAG = 'latest'
        DOCKERFILE_NAME = 'Dockerfile.multistage'
        DATABASE = 'shoppinglist'
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
                withCredentials([
                    usernamePassword(credentialsId: 'shopping-list-database', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD')
                ]) {
                    // Injecting credentials into the config files
                    sh 'cp setup/dbConfig.template.json resources/dockerDb.json'
                    def dockerImage = docker.build("${env.DOCKER_IMAGE}:${env.DOCKER_TAG}", "-f ${env.DOCKERFILE_NAME} .")
                }
            }
        }
        stage('Deploy') {
            steps {
                script {
                    docker.withRegistry('https://index.docker.io/v1/', 'dockerhub-access-token') {
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