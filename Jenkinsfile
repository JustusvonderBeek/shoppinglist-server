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
                sh 'go test'
            }
        }
        stage('Deploy') {
            steps {
                echo 'Deploying....'
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