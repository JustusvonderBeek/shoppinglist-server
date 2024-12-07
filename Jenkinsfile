pipeline {
    agent any

    tools {
        go '1.20'
    }
    stages {
        stage('Build') {
            steps {
                echo 'Building..'
                sh 'go build -o app-server'
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
        stage('Run') {
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