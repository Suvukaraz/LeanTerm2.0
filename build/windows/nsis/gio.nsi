Unicode true

####
## Installer of PureTerm (cmd/pureterm) for Windows, amd64.
## build/package-windows.sh calls it with:
##   makensis -DINFO_PRODUCTVERSION=2.1.3 -DARG_BINARY=pureterm.exe -DARG_PUREEDIT=pureedit.exe -DOUTFILE=setup.exe gio.nsi
## The installer is per machine (Program Files, administrator rights). The
## uninstall key keeps its name, so an update replaces an installed version.
####

!ifndef INFO_PRODUCTVERSION
    !define INFO_PRODUCTVERSION "2.1.3"
!endif
!ifndef ARG_BINARY
    !error "ARG_BINARY must name the pureterm.exe to install"
!endif
!ifndef ARG_PUREEDIT
    !error "ARG_PUREEDIT must name the pureedit.exe to install"
!endif
!ifndef OUTFILE
    !define OUTFILE "..\..\..\bin\pureterm_setup.exe"
!endif

!define INFO_PRODUCTNAME "PureTerm"
!define INFO_COMPANYNAME "PureTerm Project"
!define INFO_COPYRIGHT   "© 2026 PureTerm Project, GPL-3.0"
!define PRODUCT_EXECUTABLE "pureterm.exe"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"

# PureEdit: PureTerm's text editor on its own, with an executable of its own
# (icon, name in "Open with" and the taskbar).
!define PUREEDIT_EXECUTABLE "pureedit.exe"
!define PUREEDIT_PROGID "PureTerm.PureEdit"
!define PUREEDIT_CMD '"$INSTDIR\${PUREEDIT_EXECUTABLE}"'
# The AppUserModelIDs of the windows (see internal/gui/winsys): shortcuts
# carry them too, so a pinned shortcut and its windows share a button.
!define PURETERM_APPID "org.pureterm.PureTerm"
!define PUREEDIT_APPID "org.pureterm.PureEdit"

RequestExecutionLevel admin

# The version information must consist of 4 parts.
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"
!include "x64.nsh"
!include "WinVer.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can read the details.
!define MUI_ABORTWARNING # Warn the user if they exit from the installer.

!insertmacro MUI_PAGE_WELCOME
!define MUI_COMPONENTSPAGE_SMALLDESC
!insertmacro MUI_PAGE_COMPONENTS # PureEdit is optional.
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
OutFile "${OUTFILE}"
InstallDir "$PROGRAMFILES64\${INFO_PRODUCTNAME}"
ShowInstDetails show

Function .onInit
    ${IfNot} ${AtLeastWin10}
        MessageBox MB_OK "PureTerm needs Windows 10 or later."
        Abort
    ${EndIf}
    ${IfNot} ${RunningX64}
        MessageBox MB_OK "PureTerm is built for 64-bit Windows only."
        Abort
    ${EndIf}
FunctionEnd

Section "PureTerm" SecPureTerm
    SectionIn RO # always installed
    SetShellVarContext all

    SetOutPath $INSTDIR
    File "/oname=${PRODUCT_EXECUTABLE}" "${ARG_BINARY}"
    File "..\pureedit.ico"

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortcut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    ExecWait '"$INSTDIR\${PRODUCT_EXECUTABLE}" -set-shortcut-appid ${PURETERM_APPID} "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$DESKTOP\${INFO_PRODUCTNAME}.lnk"'

    WriteUninstaller "$INSTDIR\uninstall.exe"
    WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
    WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1
SectionEnd

####
## PureEdit: its own shortcuts and "Open with" for text files.
####
!macro pureedit.openWith EXT
    WriteRegStr HKLM "Software\Classes\${EXT}\OpenWithProgids" "${PUREEDIT_PROGID}" ""
!macroend
!macro pureedit.removeOpenWith EXT
    DeleteRegValue HKLM "Software\Classes\${EXT}\OpenWithProgids" "${PUREEDIT_PROGID}"
!macroend
# Text file types offered "Open with PureEdit" (used for install and uninstall).
!macro pureedit.forEachExtension MACRO
    !insertmacro ${MACRO} ".txt"
    !insertmacro ${MACRO} ".log"
    !insertmacro ${MACRO} ".md"
    !insertmacro ${MACRO} ".ini"
    !insertmacro ${MACRO} ".cfg"
    !insertmacro ${MACRO} ".conf"
    !insertmacro ${MACRO} ".config"
    !insertmacro ${MACRO} ".json"
    !insertmacro ${MACRO} ".xml"
    !insertmacro ${MACRO} ".yaml"
    !insertmacro ${MACRO} ".yml"
    !insertmacro ${MACRO} ".toml"
    !insertmacro ${MACRO} ".csv"
    !insertmacro ${MACRO} ".properties"
    !insertmacro ${MACRO} ".env"
    !insertmacro ${MACRO} ".sql"
    !insertmacro ${MACRO} ".sh"
    !insertmacro ${MACRO} ".ps1"
    !insertmacro ${MACRO} ".bat"
    !insertmacro ${MACRO} ".cmd"
    !insertmacro ${MACRO} ".py"
    !insertmacro ${MACRO} ".js"
    !insertmacro ${MACRO} ".ts"
    !insertmacro ${MACRO} ".go"
    !insertmacro ${MACRO} ".c"
    !insertmacro ${MACRO} ".h"
    !insertmacro ${MACRO} ".cpp"
    !insertmacro ${MACRO} ".cs"
    !insertmacro ${MACRO} ".java"
    !insertmacro ${MACRO} ".css"
    !insertmacro ${MACRO} ".html"
    !insertmacro ${MACRO} ".htm"
!macroend

Section "PureEdit (text editor)" SecPureEdit
    SetShellVarContext all

    SetOutPath $INSTDIR
    File "/oname=${PUREEDIT_EXECUTABLE}" "${ARG_PUREEDIT}"

    CreateShortcut "$SMPROGRAMS\PureEdit.lnk" "$INSTDIR\${PUREEDIT_EXECUTABLE}" "" "$INSTDIR\${PUREEDIT_EXECUTABLE}" 0 "" "" "PureEdit text editor"
    CreateShortcut "$DESKTOP\PureEdit.lnk" "$INSTDIR\${PUREEDIT_EXECUTABLE}" "" "$INSTDIR\${PUREEDIT_EXECUTABLE}" 0 "" "" "PureEdit text editor"
    ExecWait '"$INSTDIR\${PRODUCT_EXECUTABLE}" -set-shortcut-appid ${PUREEDIT_APPID} "$SMPROGRAMS\PureEdit.lnk" "$DESKTOP\PureEdit.lnk"'

    # ProgID used by "Open with": name and icon of PureEdit, opens the file in it.
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}" "" "Text Document (PureEdit)"
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}" "FriendlyTypeName" "Text Document (PureEdit)"
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}" "AppUserModelID" "${PUREEDIT_APPID}"
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}\DefaultIcon" "" "$INSTDIR\${PUREEDIT_EXECUTABLE},0"
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}\Application" "ApplicationName" "PureEdit"
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}\Application" "ApplicationDescription" "PureEdit text editor (PureTerm)"
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}\Application" "ApplicationIcon" "$INSTDIR\${PUREEDIT_EXECUTABLE},0"
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}\Application" "AppUserModelID" "${PUREEDIT_APPID}"
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}\shell\open" "FriendlyAppName" "PureEdit"
    WriteRegStr HKLM "Software\Classes\${PUREEDIT_PROGID}\shell\open\command" "" '${PUREEDIT_CMD} "%1"'
    !insertmacro pureedit.forEachExtension pureedit.openWith
    # The application itself, for "Open with" → "Choose another app".
    WriteRegStr HKLM "Software\Classes\Applications\${PUREEDIT_EXECUTABLE}" "FriendlyAppName" "PureEdit"
    WriteRegStr HKLM "Software\Classes\Applications\${PUREEDIT_EXECUTABLE}\DefaultIcon" "" "$INSTDIR\${PUREEDIT_EXECUTABLE},0"
    WriteRegStr HKLM "Software\Classes\Applications\${PUREEDIT_EXECUTABLE}\shell\open\command" "" '${PUREEDIT_CMD} "%1"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\App Paths\${PUREEDIT_EXECUTABLE}" "" "$INSTDIR\${PUREEDIT_EXECUTABLE}"
    # Any other file Windows considers text (perceived type "text").
    WriteRegStr HKLM "Software\Classes\SystemFileAssociations\text\OpenWithProgids" "${PUREEDIT_PROGID}" ""

    # Tell Explorer about the new "Open with" entries.
    System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecPureTerm} "PureTerm connection manager (required)."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecPureEdit} "PureEdit, PureTerm's text editor on its own: Start menu and desktop shortcut and $\"Open with$\" for text files."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "un.Uninstall"
    SetShellVarContext all

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    # PureEdit (nothing to do when it was not selected)
    Delete "$SMPROGRAMS\PureEdit.lnk"
    Delete "$DESKTOP\PureEdit.lnk"
    !insertmacro pureedit.forEachExtension pureedit.removeOpenWith
    DeleteRegValue HKLM "Software\Classes\SystemFileAssociations\text\OpenWithProgids" "${PUREEDIT_PROGID}"
    DeleteRegKey HKLM "Software\Classes\${PUREEDIT_PROGID}"
    DeleteRegKey HKLM "Software\Classes\Applications\${PUREEDIT_EXECUTABLE}"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\App Paths\${PUREEDIT_EXECUTABLE}"
    System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'

    RMDir /r $INSTDIR

    DeleteRegKey HKLM "${UNINST_KEY}"
SectionEnd
