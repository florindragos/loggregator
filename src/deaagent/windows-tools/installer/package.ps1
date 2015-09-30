<#
.SYNOPSIS
    Packaging and installation script for Windows Dea Agent.
.DESCRIPTION
    This script packages all the Dea Agent binaries into an self-extracting file.
    Upon self-extraction this script is run to unpack and install the Dea Agent service.
.PARAMETER action
    This is the parameter that specifies what the script should do: package the binaries and create the installer, or install the services.
.PARAMETER binDir
    When the action is 'package', this parameter specifies where the Dea Agent binaries are located. Not used otherwise.
.NOTES
    Author: Vlad Iovanov
    Date:   September 10, 2015
#>
param (
    [Parameter(Mandatory=$true)]
    [ValidateSet('package','install')]
    [string] $action,
    [string] $binDir
)

if (($pshome -like "*syswow64*") -and ((Get-WmiObject Win32_OperatingSystem).OSArchitecture -like "64*")) {
    Write-Warning "Restarting script under 64 bit powershell"
    
    $powershellLocation = join-path ($pshome -replace "syswow64", "sysnative") "powershell.exe"
    $scriptPath = $SCRIPT:MyInvocation.MyCommand.Path
    
    # relaunch this script under 64 bit shell
    $process = Start-Process -Wait -PassThru -NoNewWindow $powershellLocation "-nologo -file ${scriptPath} -action ${action} -binDir ${binDir}"
    
    # This will exit the original powershell process. This will only be done in case of an x86 process on a x64 OS.
    exit $process.ExitCode
}

# Helper for Null coalescing
function Null-Coalesce($a, $b)
{ 
    if ([string]::IsNullOrWhiteSpace($a)) 
    { 
        return $b
    } 
    else 
    { 
        return $a
    } 
}

# Entry point of the script when the action is "package"
function DoAction-Package($binDir)
{
    Write-Output "Packaging files from the ${binDir} dir ..."
    [Reflection.Assembly]::LoadWithPartialName( "System.IO.Compression.FileSystem" ) | out-null

    $destFile = Join-Path $(Get-Location) "binaries.zip"
    $compressionLevel = [System.IO.Compression.CompressionLevel]::Optimal
    $includeBaseDir = $false
    Remove-Item -Force -Path $destFile -ErrorAction SilentlyContinue

    Write-Output 'Creating zip ...'

    [System.IO.Compression.ZipFile]::CreateFromDirectory($binDir, $destFile, $compressionLevel, $includeBaseDir)

    Write-Output 'Creating the self extracting exe ...'

    $installerProcess = Start-Process -Wait -PassThru -NoNewWindow 'iexpress' "/N /Q deaagent-installer.sed"

    if ($installerProcess.ExitCode -ne 0)
    {
        Write-Error "There was an error building the installer."
        exit 1
    }
    
    Write-Output 'Removing artifacts ...'
    Remove-Item -Force -Path $destfile -ErrorAction SilentlyContinue
    
    Write-Output 'Done.'
}

# Entry point of the script when the action is "install"
function DoAction-Install()
{
	Write-Output 'Stopping existing Dea Agent service'
	Stop-Service -Name "deaagent" -ErrorAction SilentlyContinue | Out-Null

    Write-Output 'Installing Dea Agent service ...'
  
    $env:METRON_HOST                               = Null-Coalesce $env:METRON_HOST                               "127.0.0.1"
    $env:METRON_DROPSONDE_INCOMING_MESSAGESPORT    = Null-Coalesce $env:METRON_DROPSONDE_INCOMING_MESSAGESPORT    "3457"
    $env:METRON_VARZ_USER                          = Null-Coalesce $env:METRON_VARZ_USER                          ""
    $env:METRON_VARZ_PASS                          = Null-Coalesce $env:METRON_VARZ_PASS                          ""
    $env:METRON_VARZ_PORT                          = Null-Coalesce $env:METRON_VARZ_PORT                          "0"
    $env:LOGGREGATOR_SHARED_SECRET                 = Null-Coalesce $env:LOGGREGATOR_SHARED_SECRET                 "loggregator-secret"
    $env:LOGGREGATOR_JOB_INDEX                     = Null-Coalesce $env:LOGGREGATOR_JOB_INDEX                     "0"
    $env:NATS_HOSTS                                = Null-Coalesce $env:NATS_HOSTS                                "nats.service.dc1.consul"
    $env:NATS_PORT                                 = Null-Coalesce $env:NATS_PORT                                 "4222"
    $env:NATS_USER                                 = Null-Coalesce $env:NATS_USER                                 "nats"
    $env:NATS_PASS                                 = Null-Coalesce $env:NATS_PASS                                 "nats"
    $env:SYSLOG                                    = Null-Coalesce $env:SYSLOG                                    ""
    $env:DEAAGENT_INSTALL_DIR                      = Null-Coalesce $env:DEAAGENT_INSTALL_DIR                      "c:\deaagent"

    Write-Output "Using METRON_HOST                               $($env:METRON_HOST)"
    Write-Output "Using METRON_DROPSONDE_INCOMING_MESSAGESPORT    $($env:METRON_DROPSONDE_INCOMING_MESSAGESPORT)"
    Write-Output "Using METRON_VARZ_USER                          $($env:METRON_VARZ_USER)"
    Write-Output "Using METRON_VARZ_PASS                          $($env:METRON_VARZ_PASS)"
    Write-Output "Using METRON_VARZ_PORT                          $($env:METRON_VARZ_PORT)"
    Write-Output "Using LOGGREGATOR_SHARED_SECRET                 $($env:LOGGREGATOR_SHARED_SECRET)"
    Write-Output "Using LOGGREGATOR_JOB_INDEX                     $($env:LOGGREGATOR_JOB_INDEX)"
    Write-Output "Using NATS_HOSTS                                $($env:NATS_HOSTS)"
    Write-Output "Using NATS_PORT                                 $($env:NATS_PORT)"
    Write-Output "Using NATS_USER                                 $($env:NATS_USER)"
    Write-Output "Using NATS_PASS                                 $($env:NATS_PASS)"
    Write-Output "Using SYSLOG                                    $($env:SYSLOG)"

    $configuration = @{}
	
    $configuration["Index"] = [int]$env:LOGGREGATOR_JOB_INDEX
    $configuration["MetronAddress"] = "$($env:METRON_HOST):$($env:METRON_DROPSONDE_INCOMING_MESSAGESPORT)"
    $configuration["SharedSecret"] = $env:LOGGREGATOR_SHARED_SECRET 
    $configuration["NatsHosts"] = @($env:NATS_HOSTS)
    $configuration["NatsPort"] = [int]$env:NATS_PORT
    $configuration["NatsUser"] = $env:NATS_USER
    $configuration["NatsPass"] = $env:NATS_PASS
    $configuration["VarzUser"] = $env:METRON_VARZ_USER 
    $configuration["VarzPass"] = $env:METRON_VARZ_PASS 
    $configuration["VarzPort"] = [int]$env:METRON_VARZ_PORT 	
	$configuration["Syslog"] = $env:SYSLOG 

    $destFolder = $env:DEAAGENT_INSTALL_DIR
    $configFolder = Join-Path $destFolder 'config'
    $logsFolder = Join-Path $destFolder 'logs'
    
    foreach ($dir in @($destFolder, $configFolder, $logsFolder))
    {
        Write-Output "Cleaning up directory ${dir}"
        Remove-Item -Force -Recurse -Path $dir -ErrorVariable errors -ErrorAction SilentlyContinue

        if ($errs.Count -eq 0)
        {
            Write-Output "Successfully cleaned the directory ${dir}"
        }
        else
        {
            Write-Error "There was an error cleaning up the directory '${dir}'.`r`nPlease make sure the folder and any of its child items are not in use, then run the installer again."
            exit 1;
        }

        Write-Output "Setting up directory ${dir}"
        New-Item -path $dir -type directory -Force -ErrorAction SilentlyContinue
    }

    [Reflection.Assembly]::LoadWithPartialName( "System.IO.Compression.FileSystem" ) | out-null
    $srcFile = ".\binaries.zip"

    Write-Output 'Unpacking files ...'
    try
    {
        [System.IO.Compression.ZipFile]::ExtractToDirectory($srcFile, $destFolder)
    }
    catch
    {
        Write-Error "There was an error writing to the installation directory '${destFolder}'.`r`nPlease make sure the folder and any of its child items are not in use, then run the installer again."
        exit 1;
    }

    InstallMetron $destfolder $configuration
}

# This function calls the nssm.exe binary to set a property
function SetNSSMParameter($serviceName, $parameterName, $parameterValue)
{
    Write-Output "Setting parameter '${parameterName}' for service '${serviceName}'"
    $nssmProcess = Start-Process -Wait -PassThru -NoNewWindow 'nssm' "set ${serviceName} ${parameterName} ${parameterValue}"

    if ($nssmProcess.ExitCode -ne 0)
    {
        Write-Error "There was an error setting the ${parameterName} NSSM parameter."
        exit 1
    }
}

# This function calls the nssm.exe binary to install a new  Windows Service
function InstallNSSMService($serviceName, $executable)
{
    Write-Output "Installing service '${serviceName}'"
    
    $nssmProcess = Start-Process -Wait -PassThru -NoNewWindow 'nssm' "remove ${serviceName} confirm"
   
    if (($nssmProcess.ExitCode -ne 0) -and ($nssmProcess.ExitCode -ne 3))
    {
        Write-Error "There was an error removing the '${serviceName}' service."
        exit 1
    }
    
    $nssmProcess = Start-Process -Wait -PassThru -NoNewWindow 'nssm' "install ${serviceName} ${executable}"

    if (($nssmProcess.ExitCode -ne 0) -and ($nssmProcess.ExitCode -ne 5))
    {
        Write-Error "There was an error installing the '${serviceName}' service."
        exit 1
    }
}

# This function sets up a Windows Service using the Non Sucking Service Manager
function SetupNSSMService($serviceName, $serviceDisplayName, $serviceDescription, $startupDirectory, $executable, $arguments, $stdoutLog, $stderrLog)
{
    InstallNSSMService $serviceName $executable
	SetNSSMParameter $serviceName "ObjectName"      "NetworkService"
    SetNSSMParameter $serviceName "DisplayName"     $serviceDisplayName
    SetNSSMParameter $serviceName "Description"     $serviceDescription
    SetNSSMParameter $serviceName "AppDirectory"    $startupDirectory
    SetNSSMParameter $serviceName "AppParameters"   $arguments
    SetNSSMParameter $serviceName "AppStdout"       $stdoutLog
    SetNSSMParameter $serviceName "AppStderr"       $stderrLog
}


# This function does all the installation. Writes the config, installs services, etc.
function InstallDeaAgent($destfolder, $configuration)
{
    Write-Output "Writing JSON configuration ..."
    $configFile = Join-Path $destFolder "config\dea_logging_agent.json"
    $configuration | ConvertTo-Json | Out-File -Encoding ascii -FilePath $configFile

    Write-Output "Installing nssm services ..."

    $serviceConfigs = @{
        "deaagent" = @{
            "serviceDisplayName" = "Loggregator Dea Agent";
            "serviceDescription" = "A component that forwards logs from applications to metron";
            "startupDirectory" = $destFolder;
            "executable" = (Join-Path $destFolder "deaagent.exe");
            "arguments" = "-config ${configFile}";
            "stdoutLog" = Join-Path $destFolder "logs\deaagent.stdout.log";
            "stderrLog" = Join-Path $destFolder "logs\deaagent.stderr.log";
        };
    }
    
    # Setup windows services
    foreach ($serviceName in $serviceConfigs.Keys)
    {
        $serviceConfig = $serviceConfigs[$serviceName]
        $serviceDisplayName = $serviceConfig["serviceDisplayName"]
        $serviceDescription = $serviceConfig["serviceDescription"]
        $startupDirectory = $serviceConfig["startupDirectory"]
        $executable = $serviceConfig["executable"]
        $arguments = $serviceConfig["arguments"]
        $stdoutLog = $serviceConfig["stdoutLog"]
        $stderrLog = $serviceConfig["stderrLog"]
        SetupNSSMService $serviceName $serviceDisplayName $serviceDescription $startupDirectory $executable $arguments $stdoutLog $stderrLog
    }
    
    # Start deaagent
    Start-Service -Name "deaagent"
}

if ($action -eq 'package')
{
    if ([string]::IsNullOrWhiteSpace($binDir))
    {
        Write-Error 'The binDir parameter is mandatory when packaging.'
        exit 1
    }
    
    $binDir = Resolve-Path $binDir
    
    if ((Test-Path $binDir) -eq $false)
    {
        Write-Error "Could not find directory ${binDir}."
        exit 1        
    }
    
    Write-Output "Using binary dir ${binDir}"
    
    DoAction-Package $binDir
}
elseif ($action -eq 'install')
{
    DoAction-Install
}