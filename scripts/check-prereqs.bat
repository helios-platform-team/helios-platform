@echo off
REM =============================================================================
REM Helios Platform - Prerequisite Checker (Windows)
REM =============================================================================
REM Verifies that all required tools are installed and properly configured.
REM Can be run standalone or via `task check` (requires Git Bash or WSL).
REM
REM Usage: scripts\check-prereqs.bat [--env]
REM   --env   Also validate that .env exists and required variables are set
REM =============================================================================
setlocal enabledelayedexpansion

set "ERRORS=0"
set "WARNINGS=0"
set "CHECK_ENV=0"

REM Parse arguments
for %%a in (%*) do (
    if "%%a"=="--env" set "CHECK_ENV=1"
)

echo.
echo ==============================
echo  Helios Platform - Prereqs
echo ==============================
echo.

REM ---------------------------------------------------------------------------
REM Core Tools
REM ---------------------------------------------------------------------------
echo [Core Tools]

call :check_tool "go" "go version"
call :check_tool "docker" "docker --version"
call :check_tool "kubectl" "kubectl version --client"
call :check_tool "k3d" "k3d version"
call :check_tool "cue" "cue version"

echo.
echo [Node.js / Frontend]

call :check_tool "node" "node --version"
call :check_tool "yarn" "yarn --version"

echo.
echo [Runtime Checks]

docker info >nul 2>&1
if %errorlevel% equ 0 (
    echo   [OK]   Docker daemon is running
) else (
    echo   [FAIL] Docker daemon is not running. Start Docker Desktop first.
    set /a ERRORS+=1
)

if exist "%USERPROFILE%\.kube\config" (
    echo   [OK]   Kubeconfig found
) else if defined KUBECONFIG (
    echo   [OK]   Kubeconfig found via KUBECONFIG env var
) else (
    echo   [WARN] No kubeconfig found. One will be created by 'task setup:cluster'.
    set /a WARNINGS+=1
)

REM ---------------------------------------------------------------------------
REM Optional: .env validation
REM ---------------------------------------------------------------------------
if %CHECK_ENV% equ 1 (
    echo.
    echo [Environment Variables]

    REM Resolve repo root relative to this script
    set "SCRIPT_DIR=%~dp0"
    set "ENV_FILE=!SCRIPT_DIR!..\.env"

    if not exist "!ENV_FILE!" (
        echo   [FAIL] .env file not found at repo root. Run: copy .env.example .env
        set /a ERRORS+=1
    ) else (
        echo   [OK]   .env file exists

        REM Source .env by parsing key=value lines
        for /f "usebackq tokens=1,* delims==" %%i in ("!ENV_FILE!") do (
            set "line=%%i"
            REM Skip comment lines
            if not "!line:~0,1!"=="#" (
                if not "%%j"=="" set "%%i=%%j"
            )
        )

        REM Check required variables
        call :check_env_var "GITHUB_TOKEN"
        call :check_env_var "GITHUB_USER"
        call :check_env_var "AUTH_GITHUB_CLIENT_ID"
        call :check_env_var "AUTH_GITHUB_CLIENT_SECRET"
    )
)

REM ---------------------------------------------------------------------------
REM Summary
REM ---------------------------------------------------------------------------
echo.
echo ==============================
if %ERRORS% gtr 0 (
    echo  %ERRORS% error^(s^) and %WARNINGS% warning^(s^).
    echo  Fix the errors above before proceeding.
    exit /b 1
) else if %WARNINGS% gtr 0 (
    echo  All required tools found. %WARNINGS% warning^(s^) to review.
    exit /b 0
) else (
    echo  All checks passed!
    exit /b 0
)

REM ---------------------------------------------------------------------------
REM Subroutines
REM ---------------------------------------------------------------------------

:check_tool
REM %~1 = tool name, %~2 = version command
where %~1 >nul 2>&1
if %errorlevel% equ 0 (
    for /f "delims=" %%v in ('%~2 2^>^&1') do (
        echo   [OK]   %~1 - %%v
        goto :eof
    )
    echo   [OK]   %~1 found
) else (
    echo   [FAIL] %~1 not found. Please install it.
    set /a ERRORS+=1
)
goto :eof

:check_env_var
REM %~1 = variable name
if not defined %~1 (
    echo   [FAIL] %~1 is not set in .env
    set /a ERRORS+=1
    goto :eof
)
set "val=!%~1!"
if "!val!"=="" (
    echo   [FAIL] %~1 is empty in .env
    set /a ERRORS+=1
) else if "!val:~0,8!"=="ghp_xxxx" (
    echo   [FAIL] %~1 still has placeholder value in .env
    set /a ERRORS+=1
) else if "!val:~0,5!"=="your-" (
    echo   [FAIL] %~1 still has placeholder value in .env
    set /a ERRORS+=1
) else (
    echo   [OK]   %~1 is configured
)
goto :eof
