#!groovy

@Library(value='jenkins-scripts@vda-ukprod', changelog=false) _

//veritonePipeline {}


    pipeline {
        agent { node { label "linux" } }

        options {
            timestamps()
            timeout(time: 90, unit: 'MINUTES')
            disableConcurrentBuilds()
            buildDiscarder(logRotator(daysToKeepStr: '14'))
        }

        stages
        {
            stage('Setup Environment')
            {
                steps {
                    script {
                        env.AWS_ACCOUNT_ID = awsAccountId()
                        env.GIT_REPO_NAME = "${GIT_URL.reverse().tokenize('/')[0].reverse()[0..-5]}"

                        ARTIFACT_TYPES = getArtifactTypes()
                        AWS_ECR_PREFIX = getEcrNamePrefix()
                        AWS_REGIONS = getAwsRegions()
                        DEPLOY_TYPES = getDeployTypes()
                        REPOSITORY_TYPES = getRepositoryTypes()
                        DOCKER_BUILD_ARGS = getDockerBuildArgs()
                        DOCKER_TAGS = getDockerTags()
                        APPLICATION_LANGUAGE = getApplicationLanguage()
                        APPLICATION_VERSION = getApplicationVersion()
                        // *** REMOVE THIS IMMEDIATELY ONCE DISCOVERY ON NODE8
                        APPLICATION_NPM_VERSION = getApplicationNpmVersion()
                        // ***
                        APPROVAL_REQUIRED = isPipelineRequireApproval()
                        DEPLOY_REQUIRED = isDeployTypes(DEPLOY_TYPES)
                        DEPLOY_DEV_ON_FEATURE = isDeployToDevOnFeature()
                        SERVICES_FOR_TERRAFORM = getServicesLinkedToRepo()
                        PULL_REQUEST_URL = getGitPullRequest(GIT_REPO_NAME, GIT_COMMIT)
                        AWS_AMIS = []
                        VDA_ENGINES = []
                    }
                }
            }
            stage('Build')
            {
                steps {
                    builder(gitUrl: GIT_URL,
                    applicationLanguage: APPLICATION_LANGUAGE,
                    applicationNpmVersion: APPLICATION_NPM_VERSION,
                    applicationVersion: APPLICATION_VERSION,
                    gitCommit: GIT_COMMIT,
                    packageName: GIT_REPO_NAME,
                    artifactTypes: ARTIFACT_TYPES,
                    dockerBuildArgs: DOCKER_BUILD_ARGS)
                }
            }
            stage('Publish')
            {
                steps {
                    // TODO: Had to wrap in script block to store AMIS as variable
                    // Find a better solution to include Declarative error handling
                    script {
                        AWS_AMIS = publisher(packageName: GIT_REPO_NAME,
                        artifactTypes: ARTIFACT_TYPES,
                        awsEcrPrefix: AWS_ECR_PREFIX,
                        repositoryTypes: REPOSITORY_TYPES,
                        awsRegions: AWS_REGIONS,
                        gitCommit: GIT_COMMIT,
                        applicationLanguage: APPLICATION_LANGUAGE,
                        applicationVersion: APPLICATION_VERSION,
                        dockerImageTags: DOCKER_TAGS)
                    }
                }
            }
            stage("Deploy \'DEV\'") {
                when {
                    beforeAgent true
                    allOf {
                        anyOf {
                            //no longer deploying to dev for master branch
                            //expression { BRANCH_NAME == 'master' }
                            expression { BRANCH_NAME.trim().toLowerCase().startsWith("feature/") }
                        }
                        expression { VERITONE_ENVIRONMENT != 'GovCloud' }
                        expression { DEPLOY_REQUIRED }
                        expression { DEPLOY_DEV_ON_FEATURE }
                    }
                }

                environment {
                    ENVIRONMENT = "dev"
                }

                steps {
                    lock("LOCK_${env.GIT_REPO_NAME}_${ENVIRONMENT}") {
                        deployer(services: SERVICES_FOR_TERRAFORM,
                        packageName: GIT_REPO_NAME,
                        awsEcrPrefix: AWS_ECR_PREFIX,
                        awsRegions: AWS_REGIONS,
                        environment: ENVIRONMENT,
                        deployTypes: DEPLOY_TYPES,
                        awsAmis: AWS_AMIS,
                        buildNumber: BUILD_NUMBER)

                        tagger(buildNumber: BUILD_NUMBER,
                        environment: ENVIRONMENT,
                        branch: GIT_BRANCH)
                    }
                }
            }
            stage("Post-Deploy Tests \'DEV\'")
            {
                when {
                    beforeAgent true
                    allOf {
                        anyOf {
                            //expression { BRANCH_NAME == 'master' }
                            expression { BRANCH_NAME.trim().toLowerCase().startsWith("feature/") }
                        }
                        expression { VERITONE_ENVIRONMENT != 'GovCloud' }
                        expression { DEPLOY_REQUIRED }
                        expression { DEPLOY_DEV_ON_FEATURE }
                    }
                }

                agent { node { label 'linux-tester' } }

                environment {
                    ENVIRONMENT = "dev"
                }

                steps {
                    tester()
                }
            }
            stage("Approve \'PROD\'")
            {
                when {
                    beforeAgent true
                    allOf {
                        expression { VERITONE_ENVIRONMENT != 'GovCloud' }
                        expression { BRANCH_NAME == 'master' }
                        expression { APPROVAL_REQUIRED }
                        expression { DEPLOY_REQUIRED }
                    }
                }

                agent none

                environment {
                    ENVIRONMENT = "prod"
                }

                steps {
                    promotionApproval(72, GIT_REPO_NAME, ENVIRONMENT, "veritone*ProdApprovers")
                }

            }
            stage("Deploy \'PROD\'") {
                when {
                    beforeAgent true
                    allOf {
                        expression { VERITONE_ENVIRONMENT != 'GovCloud' }
                        expression { BRANCH_NAME == 'master' }
                        expression { DEPLOY_REQUIRED }
                    }
                }

                environment {
                    ENVIRONMENT = "prod"
                }

                steps {
                    lock("LOCK_${env.GIT_REPO_NAME}_${ENVIRONMENT}") {
                        deployer(services: SERVICES_FOR_TERRAFORM,
                        packageName: GIT_REPO_NAME,
                        awsEcrPrefix: AWS_ECR_PREFIX,
                        awsRegions: AWS_REGIONS,
                        environment: ENVIRONMENT,
                        deployTypes: DEPLOY_TYPES,
                        awsAmis: AWS_AMIS,
                        buildNumber: BUILD_NUMBER)

                        tagger(buildNumber: BUILD_NUMBER,
                        environment: ENVIRONMENT,
                        branch: GIT_BRANCH)

                        script {
                            // https://issues.jenkins-ci.org/browse/JENKINS-49014
                            // https://github.com/jenkinsci/workflow-support-plugin/pull/52
                            // persists build log for this run if it deploys to production
                            currentBuild.rawBuild.keepLog(true)
                        }
                    }
                }
            }
            stage("Post-Deploy Tests \'PROD\'")
            {
                when {
                    beforeAgent true
                    allOf {
                        expression { VERITONE_ENVIRONMENT != 'GovCloud' }
                        expression { BRANCH_NAME == 'master' }
                        expression { DEPLOY_REQUIRED }
                    }
                }

                agent { node { label 'linux-tester' } }

                environment {
                    ENVIRONMENT = "prod"
                }

                steps {
                    tester()
                }
            }
        }
    }