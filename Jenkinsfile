pipeline {
  agent { label 'linux' }
  environment {
    HARBOR_REGISTRY = 'harbor.lab:8080'
    HARBOR_PROJECT  = 'library'
    IMAGE_NAME      = 'corona-go'
    IMAGE_TAG       = "${env.BUILD_NUMBER}"
    FULL_IMAGE      = "${HARBOR_REGISTRY}/${HARBOR_PROJECT}/${IMAGE_NAME}:${IMAGE_TAG}"
    NEXUS_BASE      = 'http://nexus.lab:8081'
    CHART_NAME      = 'corona-go'
    CHART_VERSION   = '0.1.0'
    SONAR_HOST      = 'http://sonarqube.lab:9000'
  }
  stages {
    stage('Checkout') {
      steps { checkout scm }
    }

    stage('Security: secret scan (gitleaks)') {
      steps {
        sh 'gitleaks detect --source . --no-banner --redact --exit-code 1'
      }
    }

    stage('Security: SAST (semgrep)') {
      steps {
        sh '''
          set +e
          semgrep --config=auto --severity=ERROR --quiet --error .
          EXIT_CODE=$?
          set -e
          [ "$EXIT_CODE" -ne 0 ] && { echo "Semgrep findings above"; exit 1; }
          echo "Semgrep: no high-severity findings."
        '''
      }
    }

    stage('Security: SCA (trivy)') {
      steps {
        sh '''
          trivy fs --severity HIGH,CRITICAL --exit-code 0 --no-progress .
          echo "Trivy scan complete (report-only)"
        '''
      }
    }

    stage('Security: code quality (sonarqube)') {
      steps {
        withCredentials([string(credentialsId: 'sonar-token', variable: 'SONAR_TOKEN')]) {
          sh '''
            sonar-scanner \
              -Dsonar.host.url=${SONAR_HOST} \
              -Dsonar.token=${SONAR_TOKEN}
          '''
        }
      }
    }

    stage('Build Docker image') {
      steps {
        sh "docker build --add-host=nexus.lab:10.146.183.167 -t ${FULL_IMAGE} --build-arg BUILD_VERSION=${IMAGE_TAG} ."
      }
    }

    stage('Push to Harbor') {
      steps {
        withCredentials([usernamePassword(credentialsId: 'harbor-creds',
                                          usernameVariable: 'HARBOR_USER',
                                          passwordVariable: 'HARBOR_PASS')]) {
          sh '''
            echo "$HARBOR_PASS" | docker login ${HARBOR_REGISTRY} -u "$HARBOR_USER" --password-stdin
            docker push ${FULL_IMAGE}
            docker logout ${HARBOR_REGISTRY}
          '''
        }
      }
    }

    stage('Fetch & template Helm chart') {
      steps {
        withCredentials([usernamePassword(credentialsId: 'nexus-creds',
                                          usernameVariable: 'NEXUS_USER',
                                          passwordVariable: 'NEXUS_PASS')]) {
          sh '''
            curl -fsSL -u "$NEXUS_USER:$NEXUS_PASS" \
              -o chart.tgz \
              ${NEXUS_BASE}/repository/helm-charts/${CHART_NAME}-${CHART_VERSION}.tgz
            tar -xzf chart.tgz
            helm template ${IMAGE_NAME} ./${CHART_NAME} --set image.tag=${IMAGE_TAG}
          '''
        }
      }
    }

    stage('Deploy') {
      steps {
        withCredentials([file(credentialsId: 'kubeconfig', variable: 'KUBECONFIG')]) {
          sh "helm upgrade --install ${IMAGE_NAME} ./${CHART_NAME} --set image.tag=${IMAGE_TAG} --namespace default"
        }
      }
    }
  }
  post {
    always {
      sh 'docker rmi ${FULL_IMAGE} || true'
      sh 'rm -rf chart.tgz corona-go/ || true'
    }
  }
}
