; Inno Setup script for TribalOutpost AutoDownload Companion
;
; Usage from CI:
;   iscc /DMyAppVersion=1.2.3 /DMyAppExeSource=path\to\tribaloutpost-adl.exe setup.iss
;
; Usage locally:
;   iscc setup.iss
;   (expects ..\..\bin\tribaloutpost-adl.exe)

#define MyAppName "TribalOutpost AutoDownload"
#define MyAppExeName "tribaloutpost-adl.exe"
#define MyAppPublisher "TribalOutpost"
#define MyAppURL "https://tribaloutpost.com"

; Allow version and source path to be overridden from the command line
#ifndef MyAppVersion
  #define MyAppVersion "0.0.0-dev"
#endif
#ifndef MyAppExeSource
  #define MyAppExeSource "..\..\bin\" + MyAppExeName
#endif

[Setup]
AppId={{A7D3F8E1-5B2C-4A9D-B6E4-8F1C3D7A2E5B}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputDir=output
OutputBaseFilename=tribaloutpost-adl-v{#MyAppVersion}-windows-amd64-setup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "{#MyAppExeSource}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"

[Tasks]
Name: "autostart"; Description: "Start automatically when you log in"; GroupDescription: "Additional options:"

[Registry]
; Autostart registry entry (only if task selected)
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "TribalOutpostAutoDL"; ValueData: """{app}\{#MyAppExeName}"""; Flags: uninsdeletevalue; Tasks: autostart

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "Launch {#MyAppName}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "{app}\{#MyAppExeName}"; Parameters: "autostart disable"; Flags: runhidden

[Code]
// Clean up the .bat startup file if it exists (from CLI autostart enable)
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  StartupPath: String;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    StartupPath := ExpandConstant('{userstartup}\tribaloutpost-adl.bat');
    if FileExists(StartupPath) then
      DeleteFile(StartupPath);
  end;
end;
