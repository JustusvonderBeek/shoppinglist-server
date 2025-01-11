pipeline {
    agent any

    tools {
        go '1.23'
    }
    stages {
//         stage('Checkout') {
//             steps {
//                 git branch: 'main', credentialsId: '586c569d-afaa-4f29-bf68-3ee273e79699', url: 'git@github.com:JustusvonderBeek/shoppinglist-server.git'
//             }
//         }
        stage('Build') {
            steps {
                echo 'Building..'
                sh 'go build -o app-server ./cmd/shopping-list-server'
            }
        }
        stage('Test') {
            steps {
                echo 'Testing..'
                dir('server') {
                    sh 'go test'
                }
                dir('database') {
                    sh 'go test'
                }
            }
        }
        stage('Deploy') {
            steps {
                echo 'Deploying....'
                // TODO
                // sh 'cp app-server /home/justus/shoppinglist-server/app-server'
                sh 'go install'
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