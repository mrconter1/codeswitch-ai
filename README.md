# codeswitch-ai

`codeswitch-ai` is a Go application designed to retrieve and display information from Wikipedia articles. This guide walks you through setting up and deploying the application both locally (using Docker and Kubernetes) and on AWS (using Amazon Elastic Kubernetes Service - EKS).

## Table of Contents
1. [Local Setup](#local-setup)
2. [Deploying to AWS EKS](#deploying-to-aws-eks)
3. [Resources](#resources)

---

### Local Setup

To run `codeswitch-ai` locally, we use Docker and a local Kubernetes cluster provided by Docker Desktop.

#### Prerequisites
- [Docker Desktop](https://www.docker.com/products/docker-desktop) with Kubernetes enabled
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) installed and configured

#### Steps

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/mrconter1/codeswitch-ai.git
   cd codeswitch-ai
   ```

2. **Build the Docker Image**:
   ```bash
   docker build -t codeswitch-ai:latest .
   ```

3. **Setup Kubernetes Configurations**:
   - We use `k8s/deployment.yaml` and `k8s/service.yaml` for Kubernetes deployment.

4. **Deploy to Kubernetes**:
   - Apply the Kubernetes configurations:
     ```bash
     kubectl apply -f k8s/
     ```
   - Confirm the deployment and service status:
     ```bash
     kubectl get pods
     kubectl get services
     ```

5. **Access the Application Locally**:
   - Access the app at `http://localhost:<NodePort>/article?title=Go_(programming_language)`, replacing `<NodePort>` with the assigned NodePort (use `kubectl get services` to find the port).

---

### Deploying to AWS EKS

To host `codeswitch-ai` on AWS, we’ll use Amazon Elastic Kubernetes Service (EKS) with `eksctl`.

#### Prerequisites
- [AWS CLI](https://aws.amazon.com/cli/) configured with an IAM user with EKS permissions
- [eksctl](https://eksctl.io/) installed
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) installed and configured

#### Steps

1. **Set Up EKS Cluster**:
   - Run `eksctl` to create a new EKS cluster:
     ```bash
     eksctl create cluster --name codeswitch-ai-cluster --region us-west-2 --nodes 1
     ```

2. **Configure `kubectl` for EKS**:
   - Connect `kubectl` to your new EKS cluster:
     ```bash
     aws eks --region us-west-2 update-kubeconfig --name codeswitch-ai-cluster
     ```

3. **Build and Push Docker Image to ECR**:
   - Go to **ECR (Elastic Container Registry)** in the AWS Console, create a repository named `codeswitch-ai`, and note the URI.
   - Tag and push your Docker image to ECR:
     ```bash
     docker tag codeswitch-ai:latest <your-ecr-repo-uri>:latest
     docker push <your-ecr-repo-uri>:latest
     ```

4. **Update Kubernetes Deployment for ECR**:
   - In `k8s/deployment.yaml`, update the image to use your ECR URI:
     ```yaml
     containers:
     - name: codeswitch-ai
       image: <your-ecr-repo-uri>:latest
     ```

5. **Deploy to EKS**:
   - Apply the updated Kubernetes configurations:
     ```bash
     kubectl apply -f k8s/
     ```

6. **Access the Application on AWS**:
   - With the `LoadBalancer` service type, AWS will assign a public IP to your application. Run:
     ```bash
     kubectl get services
     ```
   - Access the application using the public IP in the output, with the URL:
     ```
     http://<external-ip>:8080/article?title=Go_(programming_language)
     ```

---

### Resources

- **Docker Desktop**: [https://www.docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop)
- **AWS CLI**: [https://aws.amazon.com/cli/](https://aws.amazon.com/cli/)
- **eksctl**: [https://eksctl.io/](https://eksctl.io/)
- **kubectl**: [https://kubernetes.io/docs/tasks/tools/install-kubectl/](https://kubernetes.io/docs/tasks/tools/install-kubectl/)