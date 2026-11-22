Unicode True
!include "MUI2.nsh"

!ifndef VERSION
  !error "VERSION is required"
!endif
!ifndef APP_EXE
  !error "APP_EXE is required"
!endif
!ifndef RELAY_EXE
  !error "RELAY_EXE is required"
!endif
!ifndef HOOKS_DIR
  !error "HOOKS_DIR is required"
!endif
!ifndef APP_LICENSE
  !error "APP_LICENSE is required"
!endif
!ifndef ESPFLASH_EXE
  !error "ESPFLASH_EXE is required"
!endif
!ifndef ESPFLASH_MIT_LICENSE
  !error "ESPFLASH_MIT_LICENSE is required"
!endif
!ifndef ESPFLASH_APACHE_LICENSE
  !error "ESPFLASH_APACHE_LICENSE is required"
!endif
!ifndef OUTFILE
  !error "OUTFILE is required"
!endif

Name "Cody"
OutFile "${OUTFILE}"
InstallDir "$LOCALAPPDATA\Programs\Cody"
RequestExecutionLevel user

!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\cody.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Start Cody now"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_LANGUAGE "English"

Section "Cody" CodyInstall
  SetShellVarContext current
  ExecWait '"$SYSDIR\taskkill.exe" /IM cody.exe /T /F' $0
  SetOutPath "$INSTDIR"
  File /oname=cody.exe "${APP_EXE}"
  File /oname=cody-linux "${RELAY_EXE}"
  SetOutPath "$INSTDIR\hooks"
  File "${HOOKS_DIR}\copilot-hooks.json"
  File "${HOOKS_DIR}\cody-forward.sh"
  File "${HOOKS_DIR}\cody-forward.ps1"
  SetOutPath "$INSTDIR"
  File /oname=LICENSE "${APP_LICENSE}"
  File /oname=espflash.exe "${ESPFLASH_EXE}"
  File /oname=LICENSE-espflash-MIT.txt "${ESPFLASH_MIT_LICENSE}"
  File /oname=LICENSE-espflash-APACHE.txt "${ESPFLASH_APACHE_LICENSE}"
  WriteUninstaller "$INSTDIR\uninstall.exe"
  CreateDirectory "$SMPROGRAMS\Cody"
  CreateShortCut "$SMPROGRAMS\Cody\Cody.lnk" "$INSTDIR\cody.exe"

  ReadRegStr $0 HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Cody"
  StrCmp $0 "" install_check_existing install_write_autostart
install_check_existing:
  ReadRegStr $0 HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Cody" "InstallLocation"
  StrCmp $0 "" install_write_autostart install_autostart_done
install_write_autostart:
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Cody" "$\"$INSTDIR\cody.exe$\""
install_autostart_done:

  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Cody" "DisplayName" "Cody"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Cody" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Cody" "Publisher" "Cody"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Cody" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Cody" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Cody" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Cody" "NoRepair" 1
SectionEnd

Section "Uninstall"
  SetShellVarContext current
  ExecWait '"$SYSDIR\taskkill.exe" /IM cody.exe /T /F' $0
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Cody"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Cody"
  Delete "$SMPROGRAMS\Cody\Cody.lnk"
  RMDir "$SMPROGRAMS\Cody"
  Delete "$INSTDIR\cody.exe"
  Delete "$INSTDIR\cody-linux"
  Delete "$INSTDIR\hooks\copilot-hooks.json"
  Delete "$INSTDIR\hooks\cody-forward.sh"
  Delete "$INSTDIR\hooks\cody-forward.ps1"
  RMDir "$INSTDIR\hooks"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\espflash.exe"
  Delete "$INSTDIR\LICENSE-espflash-MIT.txt"
  Delete "$INSTDIR\LICENSE-espflash-APACHE.txt"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
SectionEnd
