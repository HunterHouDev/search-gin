@echo off
REM Search-Gin Project Build Script for Windows
REM 使用方法: build.bat [命令] [选项]

setlocal enabledelayedexpansion

set "RED=[91m"
set "GREEN=[92m"
set "YELLOW=[93m"
set "BLUE=[94m"
set "CYAN=[96m"
set "NC=[0m"

echo.
echo %CYAN%===============================================================
echo           Search-Gin 统一构建系统 (Windows)
echo ===============================================================
echo.%NC%

REM 打印函数
set "PRINT_STEP=echo %BLUE%[步骤]%NC%"
set "PRINT_SUCCESS=echo %GREEN%[成功]%NC%"
set "PRINT_ERROR=echo %RED%[错误]%NC%"
set "PRINT_WARNING=echo %YELLOW%[警告]%NC%"
set "PRINT_INFO=echo %CYAN%[信息]%NC%"

REM 检查命令是否存在
set "CHECK_CMD=where"

REM 检查 Go 环境
:check_go
%PRINT_STEP% 检查 Go 环境...
where go >nul 2>&1
if errorlevel 1 (
    %PRINT_ERROR% Go 未安装. 请先安装 Go (https://golang.org/dl/)
    exit /b 1
)
go version
%PRINT_SUCCESS% Go 环境检查完成
goto :EOF

REM 检查 Node 环境
:check_node
%PRINT_STEP% 检查 Node.js 环境...
where node >nul 2>&1
if errorlevel 1 (
    %PRINT_ERROR% Node.js 未安装. 请先安装 Node.js (https://nodejs.org/)
    exit /b 1
)
node --version
where npm >nul 2>&1
if not errorlevel 1 (
    npm --version
)
%PRINT_SUCCESS% Node.js 环境检查完成
goto :EOF

REM 初始化子模块
:init
call :check_go
call :check_node

%PRINT_STEP% 初始化子模块...

if exist "gosrc" (
    cd gosrc
    call go mod download
    call go mod tidy
    cd ..
    %PRINT_SUCCESS% Go 依赖初始化完成
)

if exist "electron_quasar" (
    cd electron_quasar
    if exist "yarn.lock" (
        call yarn install
    ) else (
        call npm install
    )
    cd ..
    %PRINT_SUCCESS% 前端依赖初始化完成
)
goto :EOF

REM 代码检查
:check
call :check_go
call :check_node

%PRINT_STEP% 代码质量检查...

if exist "gosrc" (
    cd gosrc
    where golangci-lint >nul 2>&1
    if not errorlevel 1 (
        call golangci-lint run ./...
    ) else (
        %PRINT_WARNING% golangci-lint 未安装, 跳过 Go 代码检查
    )
    call go vet ./...
    cd ..
)

%PRINT_SUCCESS% 代码质量检查完成
goto :EOF

REM 运行测试
:test
call :check_go
call :check_node

%PRINT_STEP% 运行测试...

if exist "gosrc" (
    cd gosrc
    %PRINT_INFO% 运行 Go 测试...
    call go test -v ./...
    cd ..
)

if exist "electron_quasar" (
    cd electron_quasar
    if exist "package.json" (
        %PRINT_INFO% 运行前端测试...
        if exist "yarn.lock" (
            call yarn test
        ) else (
            call npm run test
        )
    )
    cd ..
)

%PRINT_SUCCESS% 测试完成
goto :EOF

REM 构建后端
:build_backend
set "BUILD_TYPE=%~1"
if "%BUILD_TYPE%"=="" set "BUILD_TYPE=default"

%PRINT_STEP% 构建后端...

cd gosrc

set "OUTPUT_DIR=..\dist"
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

if "%BUILD_TYPE%"=="web" (
    %PRINT_INFO% 构建 Web 版本...
    call go build -o "%OUTPUT_DIR%\search-gin-web.exe" -ldflags "-H=windowsgui" .
    %PRINT_SUCCESS% Web 版本构建完成: %OUTPUT_DIR%\search-gin-web.exe
) else if "%BUILD_TYPE%"=="console" (
    %PRINT_INFO% 构建控制台版本...
    call go build -o "%OUTPUT_DIR%\search-gin-console.exe" .
    %PRINT_SUCCESS% 控制台版本构建完成: %OUTPUT_DIR%\search-gin-console.exe
) else (
    call go build -o "%OUTPUT_DIR%\search-gin.exe" .
    %PRINT_SUCCESS% 后端构建完成: %OUTPUT_DIR%\search-gin.exe
)

cd ..
goto :EOF

REM 构建前端
:build_frontend
set "BUILD_TYPE=%~1"
if "%BUILD_TYPE%"=="" set "BUILD_TYPE=default"

%PRINT_STEP% 构建前端...

cd electron_quasar

if "%BUILD_TYPE%"=="electron" (
    %PRINT_INFO% 构建 Electron 版本...
    if exist "yarn.lock" (
        call yarn topc
    ) else (
        call npm run topc
    )
    %PRINT_SUCCESS% Electron 版本构建完成
) else (
    %PRINT_INFO% 构建 Web 版本...
    if exist "yarn.lock" (
        call yarn build
    ) else (
        call npm run build
    )
    %PRINT_SUCCESS% Web 版本构建完成
)

cd ..
goto :EOF

REM 构建项目
:build
call :check_go
call :check_node
set "BUILD_TYPE=%~2"
if "%BUILD_TYPE%"=="" set "BUILD_TYPE=default"

call :build_backend "%BUILD_TYPE%"
call :build_frontend "%BUILD_TYPE%"
goto :EOF

REM 清理
:clean
%PRINT_STEP% 清理构建文件...

if exist "dist" rmdir /s /q "dist"
if exist "release" rmdir /s /q "release"

if exist "gosrc" (
    cd gosrc
    if exist ".testcoverage" rmdir /s /q ".testcoverage"
    if exist "coverage.out" del /q "coverage.out"
    if exist "coverage.html" del /q "coverage.html"
    cd ..
)

if exist "electron_quasar" (
    cd electron_quasar
    if exist "dist" rmdir /s /q "dist"
    if exist ".quasar" rmdir /s /q ".quasar"
    cd ..
)

%PRINT_SUCCESS% 清理完成
goto :EOF

REM 显示帮助
:help
echo.
echo ================================================================
echo           Search-Gin 统一构建系统
echo ================================================================
echo.
echo 用法: build.bat [命令] [选项]
echo.
echo 命令:
echo   init           初始化项目依赖
echo   check          代码质量检查
echo   test           运行测试
echo   build          构建项目
echo   build:backend  仅构建后端
echo   build:frontend 仅构建前端
echo   clean          清理构建文件
echo   help           显示帮助信息
echo.
echo 选项 ^(用于 build 命令^):
echo   web       Web 版本
echo   console   控制台版本
echo   electron  Electron 版本
echo.
echo 示例:
echo   build.bat init                 初始化项目
echo   build.bat check                代码检查
echo   build.bat test                 运行测试
echo   build.bat build                构建项目
echo   build.bat build:backend web    构建后端 Web 版本
echo   build.bat build:frontend electron 构建前端 Electron 版本
echo   build.bat clean                清理
echo.
goto :EOF

REM 主函数
:main
set "COMMAND=%~1"

if "%COMMAND%"=="" goto :help
if "%COMMAND%"=="init" goto :init
if "%COMMAND%"=="check" goto :check
if "%COMMAND%"=="test" goto :test
if "%COMMAND%"=="build" goto :build
if "%COMMAND%"=="build:backend" goto :build_backend
if "%COMMAND%"=="build:frontend" goto :build_frontend
if "%COMMAND%"=="clean" goto :clean
if "%COMMAND%"=="help" goto :help
if "%COMMAND%"=="--help" goto :help
if "%COMMAND%"=="-h" goto :help

%PRINT_ERROR% 未知命令: %COMMAND%
echo.
goto :help

endlocal
