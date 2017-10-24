pipeline {
  agent any

  stages {
    stage("Build Docker Image") {
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
