pipeline {
  agent any

  stages {
    stage("Build Docker Image: roer") {
      when {
        expression {
          return env.BRANCH_NAME == "dev"
        }
      }
      steps {
        sh("docker build -t roer .")
      }
    }
  }
}
