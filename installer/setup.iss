#ifndef AppVersion
#define AppVersion "0.1.0"
#endif
[Setup]
AppId={{CBAD5778-D9C0-4A35-A6C1-A2F369741DBB}
AppName=Pulse
AppVersion={#AppVersion}
DefaultDirName={localappdata}\Programs\Pulse
DefaultGroupName=Pulse
PrivilegesRequired=lowest
OutputDir=..\dist
OutputBaseFilename=setup
Compression=lzma2
SolidCompression=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
CloseApplications=yes
[Files]
Source: "..\dist\pulse-client.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\pulse-server.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"
[Icons]
Name: "{group}\Pulse"; Filename: "{app}\pulse-client.exe"
Name: "{autodesktop}\Pulse"; Filename: "{app}\pulse-client.exe"; Tasks: desktopicon
Name: "{group}\Pulse Server"; Filename: "{app}\pulse-server.exe"
[Tasks]
Name: desktopicon; Description: "Create desktop shortcut"
[Run]
Filename: "{app}\pulse-client.exe"; Description: "Launch Pulse"; Flags: nowait postinstall skipifsilent
