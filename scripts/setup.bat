@echo off
setlocal enabledelayedexpansion

:: Check if Claude API key was provided
if "%1"=="" (
    echo Error: Please provide Claude API key
    echo Usage: setup.bat your_claude_api_key
    exit /b 1
)

echo.
echo 🚀 Setting up CodeSwitch AI development environment...
echo.

:: Check Docker Desktop
echo 📦 Checking Docker...
docker --version > nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Docker not found. Please install Docker Desktop first.
    echo Download from: https://www.docker.com/products/docker-desktop/
    exit /b 1
)

:: Check/Install kubectl
echo 📦 Checking kubectl...
kubectl version --client > nul 2>&1
if %errorlevel% neq 0 (
    echo Installing kubectl...
    winget install -e --id Kubernetes.kubectl
    if !errorlevel! neq 0 (
        echo ❌ Failed to install kubectl
        exit /b 1
    )
)

:: Check/Install minikube
echo 📦 Checking minikube...
minikube version > nul 2>&1
if %errorlevel% neq 0 (
    echo Installing minikube...
    winget install -e --id Kubernetes.minikube
    if !errorlevel! neq 0 (
        echo ❌ Failed to install minikube
        exit /b 1
    )
)

:: Start minikube if not running
echo 🔄 Starting minikube...
minikube status > nul 2>&1
if %errorlevel% neq 0 (
    minikube start --driver=docker
    if !errorlevel! neq 0 (
        echo ❌ Failed to start minikube
        exit /b 1
    )
)

:: Clean up any existing deployment
echo 🧹 Cleaning up existing deployment...
kubectl delete namespace codeswitch 2>nul
timeout /t 5 /nobreak > nul

:: Build application
echo 🔨 Building application...
go mod tidy
if %errorlevel% neq 0 (
    echo ❌ Failed to tidy Go modules
    exit /b 1
)

:: Point to minikube's Docker daemon
echo 🔄 Configuring Docker environment...
FOR /f "tokens=*" %%i IN ('minikube docker-env') DO @%%i

:: Build Docker image
echo 🐳 Building Docker image...
docker build -t codeswitch-ai:latest .
if %errorlevel% neq 0 (
    echo ❌ Failed to build Docker image
    exit /b 1
)

:: Create namespace and deploy
echo 🚀 Deploying to Kubernetes...
kubectl create namespace codeswitch
if %errorlevel% neq 0 (
    echo ❌ Failed to create namespace
    exit /b 1
)

:: Create secret
echo 🔑 Creating secrets...
kubectl create secret generic codeswitch-secrets ^
    --namespace codeswitch ^
    --from-literal=claude-api-key=%1
if %errorlevel% neq 0 (
    echo ❌ Failed to create secrets
    exit /b 1
)

:: Apply Kubernetes configurations
echo 📦 Applying Kubernetes configurations...
kubectl apply -f k8s/deployment.yaml
if %errorlevel% neq 0 (
    echo ❌ Failed to apply deployment configuration
    exit /b 1
)

kubectl apply -f k8s/service.yaml
if %errorlevel% neq 0 (
    echo ❌ Failed to apply service configuration
    exit /b 1
)

:: Wait for pods
echo ⏳ Waiting for pods to be ready...
kubectl wait --namespace codeswitch ^
    --for=condition=ready pod ^
    --selector=app ^
    --timeout=300s
if %errorlevel% neq 0 (
    echo ❌ Timeout waiting for pods to be ready
    echo Running 'kubectl get pods -n codeswitch' for debugging:
    kubectl get pods -n codeswitch
    exit /b 1
)

echo.
echo ✅ Setup complete! Your CodeSwitch AI environment is ready.
echo.
echo 🔍 To check status:
echo    kubectl get pods -n codeswitch
echo.
echo 🧪 To test the service:
echo    kubectl port-forward -n codeswitch service/gateway 8080:8080
echo    go run cmd/test/main.go -title="Albert_Einstein" -percent=50
echo.
echo 📊 To monitor RabbitMQ:
echo    kubectl port-forward -n codeswitch service/rabbitmq-service 15672:15672
echo    Open http://localhost:15672 (guest/guest)
echo.

endlocal 