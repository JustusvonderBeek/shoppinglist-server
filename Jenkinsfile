pipeline {
    agent any

    stages {
        // Since this is a pipeline, checkout of the repository happens automatically
        stage('Build') {
            steps {
                echo 'Building..'
                sh 'go build -o app-server ./cmd/shopping-list-server'
            }
        }
        stage('Test') {
            steps {
                echo 'Testing..'
                sh 'go test ./...'
            }
        }
//         stage('Deploy') {
//             steps {
//                 echo 'Deploying....'
//                 // TODO
//                 // sh 'cp app-server /home/justus/shoppinglist-server/app-server'
//                 sh 'go install'
//             }
//         }
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