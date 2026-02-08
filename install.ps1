#Requires -Version 5.1
<#
.SYNOPSIS
    Mehrhof Windows Install Script - Installs Mehrhof inside WSL

.DESCRIPTION
    This script verifies WSL2 is configured, then installs Mehrhof inside your
    WSL Linux distribution using the standard install.sh script.

    Prerequisites:
    - Windows 10 Build 19041 or later (or Windows 11)
    - WSL2 with a Linux distribution installed (Ubuntu recommended)
    - PowerShell 5.1 or later

.PARAMETER Nightly
    Install the latest nightly build instead of stable release.

.PARAMETER Version
    Install a specific version (e.g., "v1.2.3").

.EXAMPLE
    irm https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.ps1 | iex

.EXAMPLE
    .\install.ps1 -Nightly

.EXAMPLE
    .\install.ps1 -Version "v1.2.3"

.LINK
    https://github.com/valksor/go-mehrhof
    https://valksor.com/docs/mehrhof/guides/windows-wsl
#>

[CmdletBinding()]
param(
    [switch]$Nightly,
    [string]$Version
)

# Runtime version check (the #Requires directive is bypassed when piped via irm | iex)
if ($PSVersionTable.PSVersion.Major -lt 5 -or
    ($PSVersionTable.PSVersion.Major -eq 5 -and $PSVersionTable.PSVersion.Minor -lt 1)) {
    Write-Host "[ERROR] PowerShell 5.1 or later required. Current: $($PSVersionTable.PSVersion)" -ForegroundColor Red
    Write-Host "Update PowerShell: https://aka.ms/powershell" -ForegroundColor Yellow
    exit 1
}

# Strict mode for better error detection
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Script constants
$InstallScriptUrl = "https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh"
$DocsUrl = "https://valksor.com/docs/mehrhof/guides/windows-wsl"
$MinWindowsBuild = 19041

# Colors for output
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Type = "Info"
    )
    switch ($Type) {
        "Info"    { Write-Host "[INFO] " -ForegroundColor Blue -NoNewline; Write-Host $Message }
        "Success" { Write-Host "[OK] " -ForegroundColor Green -NoNewline; Write-Host $Message }
        "Warning" { Write-Host "[WARN] " -ForegroundColor Yellow -NoNewline; Write-Host $Message }
        "Error"   { Write-Host "[ERROR] " -ForegroundColor Red -NoNewline; Write-Host $Message }
    }
}

function Show-Banner {
    Write-Host ""
    Write-Host "  __  __      _          _            __ " -ForegroundColor Cyan
    Write-Host " |  \/  | ___| |__  _ __| |__   ___  / _|" -ForegroundColor Cyan
    Write-Host " | |\/| |/ _ \ '_ \| '__| '_ \ / _ \| |_ " -ForegroundColor Cyan
    Write-Host " | |  | |  __/ | | | |  | | | | (_) |  _|" -ForegroundColor Cyan
    Write-Host " |_|  |_|\___|_| |_|_|  |_| |_|\___/|_|  " -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Structured Creation Environment - Windows Installer"
    Write-Host ""
}

function Test-WindowsBuild {
    $build = [System.Environment]::OSVersion.Version.Build
    if ($build -lt $MinWindowsBuild) {
        Write-ColorOutput "Windows build $build is too old. WSL2 requires build $MinWindowsBuild or later." -Type "Error"
        Write-Host ""
        Write-Host "To update Windows:"
        Write-Host "  1. Open Settings > Update & Security > Windows Update"
        Write-Host "  2. Check for updates and install any available"
        Write-Host ""
        exit 1
    }
    Write-ColorOutput "Windows build $build (OK)" -Type "Success"
}

function Test-WSLInstalled {
    # Check if wsl.exe exists
    $wslPath = Get-Command wsl.exe -ErrorAction SilentlyContinue
    if (-not $wslPath) {
        Write-ColorOutput "WSL is not installed." -Type "Error"
        Write-Host ""
        Write-Host "To install WSL:"
        Write-Host "  1. Open PowerShell as Administrator"
        Write-Host "  2. Run: wsl --install"
        Write-Host "  3. Restart your computer"
        Write-Host "  4. Run this script again"
        Write-Host ""
        Write-Host "For more information: $DocsUrl"
        Write-Host ""
        exit 1
    }
    Write-ColorOutput "WSL is installed" -Type "Success"
}

function Test-WSLDistribution {
    # Check if any distribution is installed
    $distros = wsl --list --quiet 2>&1

    # Handle error output (no distros installed)
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($distros) -or $distros -match "no installed distributions") {
        Write-ColorOutput "No WSL distribution installed." -Type "Error"
        Write-Host ""
        Write-Host "To install Ubuntu (recommended):"
        Write-Host "  Option 1: wsl --install -d Ubuntu"
        Write-Host "  Option 2: Install from Microsoft Store: https://aka.ms/wslstore"
        Write-Host ""
        Write-Host "After installation:"
        Write-Host "  1. Launch Ubuntu from Start menu to complete setup"
        Write-Host "  2. Create your Linux username and password"
        Write-Host "  3. Run this script again"
        Write-Host ""
        exit 1
    }

    # Get first non-empty distro name
    $firstDistro = ($distros -split "`n" | Where-Object { $_ -and $_.Trim() } | Select-Object -First 1).Trim()

    # Remove any special characters (like default markers)
    $firstDistro = $firstDistro -replace '\s*\(Default\)', '' -replace '[^\w-]', ''

    if ([string]::IsNullOrWhiteSpace($firstDistro)) {
        Write-ColorOutput "Could not detect WSL distribution name." -Type "Warning"
        $firstDistro = "your distribution"
    }

    Write-ColorOutput "WSL distribution found: $firstDistro" -Type "Success"
    return $firstDistro
}

function Install-Mehrhof {
    param(
        [string]$Distro
    )

    # Build install command
    $installArgs = ""
    if ($Nightly) {
        $installArgs = " -s -- --nightly"
    } elseif ($Version) {
        $installArgs = " -s -- -v $Version"
    }

    $installCommand = "curl -fsSL $InstallScriptUrl | bash$installArgs"

    Write-ColorOutput "Installing Mehrhof inside WSL..." -Type "Info"
    Write-Host ""
    Write-Host "Running: $installCommand" -ForegroundColor DarkGray
    Write-Host ""

    # Execute install script inside WSL
    wsl -e bash -c $installCommand

    if ($LASTEXITCODE -ne 0) {
        Write-ColorOutput "Installation failed. Check the error messages above." -Type "Error"
        exit 1
    }
}

function Test-Installation {
    Write-Host ""
    Write-ColorOutput "Verifying installation..." -Type "Info"

    $result = wsl -e mehr version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host $result
        Write-ColorOutput "Installation successful!" -Type "Success"
        return $true
    } else {
        Write-ColorOutput "Could not verify installation. You may need to restart WSL." -Type "Warning"
        Write-Host "Try: wsl --shutdown && wsl -e mehr version"
        return $false
    }
}

function Show-NextSteps {
    Write-Host ""
    Write-Host "=" * 60 -ForegroundColor Cyan
    Write-Host "  Next Steps" -ForegroundColor Cyan
    Write-Host "=" * 60 -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  1. Open WSL terminal (type 'wsl' or 'ubuntu' in PowerShell)"
    Write-Host ""
    Write-Host "  2. Navigate to your project:"
    Write-Host "     cd ~/projects/my-project"
    Write-Host ""
    Write-Host "  3. Start using Mehrhof:"
    Write-Host "     mehr --help"
    Write-Host "     mehr serve --open"
    Write-Host ""
    Write-Host "  TIP: For best performance, keep projects in the Linux"
    Write-Host "       filesystem (~/projects) rather than /mnt/c/..."
    Write-Host ""
    Write-Host "  Documentation: $DocsUrl"
    Write-Host ""
}

# Main execution
function Main {
    Show-Banner

    Write-ColorOutput "Checking prerequisites..." -Type "Info"
    Write-Host ""

    # Check Windows build version
    Test-WindowsBuild

    # Check WSL is installed
    Test-WSLInstalled

    # Check a distribution is available
    $distro = Test-WSLDistribution

    Write-Host ""

    # Install Mehrhof
    Install-Mehrhof -Distro $distro

    # Verify installation
    Test-Installation

    # Show next steps
    Show-NextSteps
}

# Run main
Main
